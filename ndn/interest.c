#include "interest.h"
#include "encode-interest.h"
#include "packet.h"
#include "tlv-encoder.h"

NdnError
PInterest_FromPacket(PInterest* interest, struct rte_mbuf* pkt,
                     struct rte_mempool* nameMp)
{
  TlvDecodePos d0;
  MbufLoc_Init(&d0, pkt);
  TlvElement interestEle;
  NdnError e = DecodeTlvElementExpectType(&d0, TT_Interest, &interestEle);
  RETURN_IF_UNLIKELY_ERROR;

  TlvDecodePos d1;
  TlvElement_MakeValueDecoder(&interestEle, &d1);
  TlvElement ele1;

#define D1_NEXT                                                                \
  do {                                                                         \
    if (MbufLoc_IsEnd(&d1)) {                                                  \
      return NdnError_OK;                                                      \
    }                                                                          \
    e = DecodeTlvElement(&d1, &ele1);                                          \
    RETURN_IF_UNLIKELY_ERROR;                                                  \
  } while (false)

  e = DecodeTlvElementExpectType(&d1, TT_Name, &ele1);
  RETURN_IF_UNLIKELY_ERROR;
  if (unlikely(ele1.length == 0)) {
    return NdnError_NameIsEmpty;
  }
  interest->name.v = TlvElement_LinearizeValue(&ele1, pkt, nameMp, &d1);
  RETURN_IF_UNLIKELY_NULL(interest->name.v, NdnError_AllocError);
  e = PName_Parse(&interest->name.p, ele1.length, interest->name.v);
  RETURN_IF_UNLIKELY_ERROR;

  interest->guiderOff = ele1.size;
  interest->guiderSize = 0;
  interest->nonce = 0;
  interest->lifetime = DEFAULT_INTEREST_LIFETIME;
  interest->hopLimit = HOP_LIMIT_OMITTED;
  interest->canBePrefix = false;
  interest->mustBeFresh = false;
  interest->nFhs = 0;
  interest->thisFhIndex = -1;

  D1_NEXT;
  if (ele1.type == TT_CanBePrefix) {
    interest->canBePrefix = true;
    interest->guiderOff += ele1.size;
    D1_NEXT;
  }

  if (ele1.type == TT_MustBeFresh) {
    interest->mustBeFresh = true;
    interest->guiderOff += ele1.size;
    D1_NEXT;
  }

  if (ele1.type == TT_ForwardingHint) {
    TlvDecodePos d2;
    TlvElement_MakeValueDecoder(&ele1, &d2);
    for (int i = 0; i < INTEREST_MAX_FHS; ++i) {
      if (MbufLoc_IsEnd(&d2)) {
        break;
      }
      TlvElement delegationEle;
      e = DecodeTlvElementExpectType(&d2, TT_Delegation, &delegationEle);
      RETURN_IF_UNLIKELY_ERROR;

      TlvDecodePos d3;
      TlvElement_MakeValueDecoder(&delegationEle, &d3);
      TlvElement ele3;
      e = DecodeTlvElementExpectType(&d3, TT_Preference, &ele3);
      RETURN_IF_UNLIKELY_ERROR;
      e = DecodeTlvElementExpectType(&d3, TT_Name, &ele3);
      interest->fh[i].value =
        TlvElement_LinearizeValue(&ele3, pkt, nameMp, &d3);
      RETURN_IF_UNLIKELY_NULL(interest->fh[i].value, NdnError_AllocError);
      interest->fh[i].length = ele3.length;
      ++interest->nFhs;
      MbufLoc_CopyPos(&d2, &d3);
    }
    MbufLoc_CopyPos(&d1, &d2);
    interest->guiderOff += ele1.size;
    D1_NEXT;
  }

  if (ele1.type == TT_Nonce) {
    if (unlikely(ele1.length != 4)) {
      return NdnError_BadNonceLength;
    }
    // overwriting ele1.value, but it's okay because we don't need it later
    rte_le32_t nonceV;
    bool ok = MbufLoc_ReadU32(&ele1.value, &nonceV);
    assert(ok); // must succeed because length is checked
    interest->nonce = rte_le_to_cpu_32(nonceV);
    interest->guiderSize += ele1.size;
    D1_NEXT;
  }

  if (ele1.type == TT_InterestLifetime) {
    uint64_t lifetimeV;
    bool ok = TlvElement_ReadNonNegativeInteger(&ele1, &lifetimeV);
    if (unlikely(!ok || lifetimeV >= UINT32_MAX)) {
      return NdnError_BadInterestLifetime;
    }
    interest->lifetime = (uint32_t)lifetimeV;
    interest->guiderSize += ele1.size;
    D1_NEXT;
  }

  if (ele1.type == TT_HopLimit) {
    if (unlikely(ele1.length != 1)) {
      return NdnError_BadHopLimitLength;
    }
    const uint8_t* hopLimitV = TlvElement_GetLinearValue(&ele1);
    if (unlikely(*hopLimitV == 0)) {
      interest->hopLimit = HOP_LIMIT_ZERO;
    } else {
      interest->hopLimit = --(*(uint8_t*)hopLimitV);
    }
    D1_NEXT;
  }

  return NdnError_OK;
#undef D1_NEXT
}

NdnError
PInterest_ParseFh(PInterest* interest, uint8_t index)
{
  assert(index < interest->nFhs);
  if (interest->thisFhIndex == index) {
    return NdnError_OK;
  }

  NdnError e = PName_Parse(&interest->thisFh.p, interest->fh[index].length,
                           interest->fh[index].value);
  RETURN_IF_UNLIKELY_ERROR;

  interest->thisFh.v = interest->fh[index].value;
  interest->thisFhIndex = index;
  return NdnError_OK;
}

Packet*
ModifyInterest(Packet* npkt, uint32_t nonce, uint32_t lifetime,
               struct rte_mempool* headerMp, struct rte_mempool* guiderMp,
               struct rte_mempool* indirectMp)
{
  assert(rte_pktmbuf_data_room_size(headerMp) >= EncodeInterest_GetHeadroom());
  assert(rte_pktmbuf_data_room_size(guiderMp) >= ModifyInterest_SizeofGuider());

  struct rte_mbuf* header = rte_pktmbuf_alloc(headerMp);
  if (unlikely(header == NULL)) {
    return NULL;
  }
  struct rte_mbuf* guider = rte_pktmbuf_alloc(guiderMp);
  if (unlikely(guider == NULL)) {
    rte_pktmbuf_free(header);
    return NULL;
  }

  struct rte_mbuf* inPkt = Packet_ToMbuf(npkt);
  PInterest* inInterest = Packet_GetInterestHdr(npkt);
  Packet* outNpkt = Packet_FromMbuf(header);

  // skip Interest TL
  TlvDecodePos d0;
  MbufLoc_Init(&d0, inPkt);
  TlvElement interestEle;
  NdnError e = DecodeTlvHeader(&d0, &interestEle);
  assert(e == NdnError_OK);

  // make indirect mbufs over Name thru ForwardingHint
  struct rte_mbuf* m1 =
    MbufLoc_MakeIndirect(&d0, inInterest->guiderOff, indirectMp);
  if (unlikely(m1 == NULL)) {
    rte_pktmbuf_free(header);
    rte_pktmbuf_free(guider);
    return NULL;
  }

  // skip old guiders
  MbufLoc_Advance(&d0, inInterest->guiderSize);

  // prepare new guiders
  TlvEncoder* enG = MakeTlvEncoder(guider);
  typedef struct GuiderF
  {
    uint8_t nonceT;
    uint8_t nonceL;
    rte_le32_t nonceV;

    uint8_t lifetimeT;
    uint8_t lifetimeL;
    rte_be32_t lifetimeV;
  } __rte_packed GuiderF;

  GuiderF* f = (GuiderF*)TlvEncoder_Append(enG, sizeof(GuiderF));
  f->nonceT = TT_Nonce;
  f->nonceL = 4;
  *(unaligned_uint32_t*)&f->nonceV = rte_cpu_to_le_32(nonce);
  f->lifetimeT = TT_InterestLifetime;
  f->lifetimeL = 4;
  *(unaligned_uint32_t*)&f->lifetimeV = rte_cpu_to_be_32(lifetime);

  // make indirect mbufs over HopLimit and Parameters, then chain after guiders
  if (d0.rem > 0) {
    struct rte_mbuf* m2 = MbufLoc_MakeIndirect(&d0, d0.rem, indirectMp);
    if (unlikely(m2 == NULL)) {
      rte_pktmbuf_free(m1);
      rte_pktmbuf_free(header);
      rte_pktmbuf_free(guider);
      return NULL;
    }

    rte_pktmbuf_chain(guider, m2);
  }

  // chain guiders after Name thru ForwardingHint
  rte_pktmbuf_chain(m1, guider);

  // prepend Interest TL
  header->data_off = header->buf_len;
  TlvEncoder* enH = MakeTlvEncoder(header);
  PrependVarNum(enH, m1->pkt_len);
  PrependVarNum(enH, TT_Interest);
  rte_pktmbuf_chain(header, m1);

  // copy LpL3 and PInterest
  L2PktType l2type = Packet_GetL2PktType(npkt);
  Packet_SetL2PktType(outNpkt, l2type);
  if (l2type == L2PktType_NdnlpV2) {
    rte_memcpy(Packet_GetLpL3Hdr(outNpkt), Packet_GetLpL3Hdr(npkt),
               sizeof(LpL3));
  }
  Packet_SetL3PktType(outNpkt, L3PktType_Interest);
  PInterest* outInterest = Packet_GetInterestHdr(outNpkt);
  rte_memcpy(outInterest, inInterest, sizeof(PInterest));
  outInterest->nonce = nonce;
  outInterest->lifetime = lifetime;
  outInterest->guiderSize = sizeof(GuiderF);

  return outNpkt;
}
