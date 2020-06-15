#include "interest.h"
#include "packet.h"
#include "tlv-encoder.h"

#include <rte_random.h>

NdnError
PInterest_FromPacket(PInterest* interest,
                     struct rte_mbuf* pkt,
                     struct rte_mempool* nameMp)
{
  MbufLoc d0;
  MbufLoc_Init(&d0, pkt);
  TlvElement interestEle;
  NdnError e = TlvElement_Decode(&interestEle, &d0, TT_Interest);
  RETURN_IF_ERROR;

  MbufLoc d1;
  TlvElement_MakeValueDecoder(&interestEle, &d1);
  TlvElement ele1;

  e = TlvElement_Decode(&ele1, &d1, TT_Name);
  RETURN_IF_ERROR;
  if (unlikely(ele1.length == 0)) {
    return NdnError_NameIsEmpty;
  }
  interest->name.v = TlvElement_LinearizeValue(&ele1, pkt, nameMp, &d1);
  RETURN_IF_NULL(interest->name.v, NdnError_AllocError);
  e = PName_Parse(&interest->name.p, ele1.length, interest->name.v);
  RETURN_IF_ERROR;

  interest->guiderOff = ele1.size;
  interest->guiderSize = 0;
  interest->nonce = 0;
  interest->lifetime = DEFAULT_INTEREST_LIFETIME;
  interest->hopLimit = 0xFF;
  interest->canBePrefix = false;
  interest->mustBeFresh = false;
  interest->nFhs = 0;
  interest->activeFh = -1;
  interest->diskData = NULL;

#define D1_NEXT                                                                \
  do {                                                                         \
    if (MbufLoc_IsEnd(&d1)) {                                                  \
      return NdnError_OK;                                                      \
    }                                                                          \
    e = TlvElement_Decode(&ele1, &d1, TT_Invalid);                             \
    RETURN_IF_ERROR;                                                           \
  } while (false)

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
    MbufLoc d2;
    TlvElement_MakeValueDecoder(&ele1, &d2);
    for (int i = 0; i < INTEREST_MAX_FHS; ++i) {
      if (MbufLoc_IsEnd(&d2)) {
        break;
      }
      TlvElement delegationEle;
      e = TlvElement_Decode(&delegationEle, &d2, TT_Delegation);
      RETURN_IF_ERROR;

      MbufLoc d3;
      TlvElement_MakeValueDecoder(&delegationEle, &d3);
      TlvElement ele3;
      e = TlvElement_Decode(&ele3, &d3, TT_Preference);
      RETURN_IF_ERROR;
      e = TlvElement_Decode(&ele3, &d3, TT_Name);
      interest->fhNameV[i] = TlvElement_LinearizeValue(&ele3, pkt, nameMp, &d3);
      RETURN_IF_NULL(interest->fhNameV[i], NdnError_AllocError);
      interest->fhNameL[i] = ele3.length;
      ++interest->nFhs;
      MbufLoc_CopyPos(&d2, &d3);
    }
    MbufLoc_CopyPos(&d1, &d2);
    interest->guiderOff += ele1.size;
    D1_NEXT;
  }

  if (ele1.type == TT_Nonce) {
    rte_le32_t nonceV;
    if (unlikely(ele1.length != sizeof(nonceV))) {
      return NdnError_BadNonceLength;
    }
    // overwriting ele1.value, but it's okay because we don't need it later
    bool ok __rte_unused = MbufLoc_ReadU32(&ele1.value, &nonceV);
    assert(ok); // must succeed because length is checked
    interest->nonce = rte_le_to_cpu_32(nonceV);
    interest->guiderSize += ele1.size;
    D1_NEXT;
  }

  if (ele1.type == TT_InterestLifetime) {
    uint64_t lifetimeV = 0;
    e = TlvElement_ReadNonNegativeInteger(&ele1, &lifetimeV);
    if (unlikely(e != NdnError_OK || lifetimeV >= UINT32_MAX)) {
      return NdnError_BadInterestLifetime;
    }
    interest->lifetime = (uint32_t)lifetimeV;
    interest->guiderSize += ele1.size;
    D1_NEXT;
  }

  if (ele1.type == TT_HopLimit) {
    if (unlikely(ele1.length != sizeof(interest->hopLimit))) {
      return NdnError_BadHopLimitLength;
    }
    const uint8_t* hopLimitV = TlvElement_GetLinearValue(&ele1);
    if (unlikely(*hopLimitV == 0)) {
      return NdnError_HopLimitZero;
    }
    interest->hopLimit = *hopLimitV;
    interest->guiderSize += ele1.size;
    D1_NEXT;
  }

  return NdnError_OK;
#undef D1_NEXT
}

NdnError
PInterest_SelectActiveFh(PInterest* interest, int8_t index)
{
  assert(index >= -1 && index < interest->nFhs);
  if (interest->activeFh == index) {
    return NdnError_OK;
  }
  interest->activeFh = -1;
  if (index < 0) {
    return NdnError_OK;
  }

  NdnError e = PName_Parse(&interest->activeFhName.p,
                           interest->fhNameL[index],
                           interest->fhNameV[index]);
  RETURN_IF_ERROR;
  interest->activeFhName.v = interest->fhNameV[index];
  interest->activeFh = index;
  return NdnError_OK;
}

void
NonceGen_Init(NonceGen* g)
{
  pcg32_srandom_r(&g->rng, rte_rand(), rte_rand());
}

Packet*
ModifyInterest(Packet* npkt,
               uint32_t nonce,
               uint32_t lifetime,
               uint8_t hopLimit,
               struct rte_mempool* headerMp,
               struct rte_mempool* guiderMp,
               struct rte_mempool* indirectMp)
{
  struct rte_mbuf* header = rte_pktmbuf_alloc(headerMp);
  if (unlikely(header == NULL)) {
    return NULL;
  }
  struct rte_mbuf* guider = rte_pktmbuf_alloc(guiderMp);
  if (unlikely(guider == NULL)) {
    rte_pktmbuf_free(header);
    return NULL;
  }
  header->data_off = header->buf_len;
  guider->data_off = 0;

  struct rte_mbuf* inPkt = Packet_ToMbuf(npkt);
  PInterest* inInterest = Packet_GetInterestHdr(npkt);
  Packet* outNpkt = Packet_FromMbuf(header);

  // skip Interest TL
  MbufLoc d0;
  MbufLoc_Init(&d0, inPkt);
  TlvElement interestEle;
  NdnError e __rte_unused = TlvElement_DecodeTL(&interestEle, &d0, TT_Interest);
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

    uint8_t hopLimitT;
    uint8_t hopLimitL;
    uint8_t hopLimitV;
  } __rte_packed GuiderF;

  GuiderF* f = (GuiderF*)TlvEncoder_Append(enG, sizeof(GuiderF));
  f->nonceT = TT_Nonce;
  f->nonceL = 4;
  *(unaligned_uint32_t*)&f->nonceV = rte_cpu_to_le_32(nonce);
  f->lifetimeT = TT_InterestLifetime;
  f->lifetimeL = 4;
  *(unaligned_uint32_t*)&f->lifetimeV = rte_cpu_to_be_32(lifetime);
  f->hopLimitT = TT_HopLimit;
  f->hopLimitL = 1;
  f->hopLimitV = hopLimit;

  // make indirect mbufs on Parameters, then chain after guiders
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
  TlvEncoder* enH = MakeTlvEncoder(header);
  PrependVarNum(enH, m1->pkt_len);
  PrependVarNum(enH, TT_Interest);
  rte_pktmbuf_chain(header, m1);

  // copy LpL3 and PInterest
  Packet_SetL2PktType(outNpkt, Packet_GetL2PktType(npkt));
  Packet_SetL3PktType(outNpkt, L3PktType_Interest);
  rte_memcpy(
    Packet_GetPriv_(outNpkt), Packet_GetPriv_(npkt), sizeof(PacketPriv));
  PInterest* outInterest = Packet_GetInterestHdr(outNpkt);
  outInterest->nonce = nonce;
  outInterest->lifetime = lifetime;
  outInterest->guiderSize = sizeof(GuiderF);
  return outNpkt;
}

void
EncodeInterest_(struct rte_mbuf* m,
                const InterestTemplate* tpl,
                uint16_t suffixL,
                const uint8_t* suffixV,
                uint32_t nonce)
{
  TlvEncoder* en = MakeTlvEncoder(m);
  AppendVarNum(en, TT_Name);
  AppendVarNum(en, tpl->prefixL + suffixL);

  uint8_t* room = TlvEncoder_Append(en, tpl->prefixL + suffixL + tpl->midLen);
  assert(room != NULL);
  rte_memcpy(room, tpl->prefixV, tpl->prefixL);
  room = RTE_PTR_ADD(room, tpl->prefixL);
  rte_memcpy(room, suffixV, suffixL);
  room = RTE_PTR_ADD(room, suffixL);
  rte_memcpy(room, tpl->midBuf, tpl->midLen);
  *(unaligned_uint32_t*)RTE_PTR_ADD(room, tpl->nonceOff) =
    rte_cpu_to_le_32(nonce);

  PrependVarNum(en, m->pkt_len);
  PrependVarNum(en, TT_Interest);
}
