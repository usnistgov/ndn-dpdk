#include "lp.h"
#include "nack.h"
#include "tlv-encoder.h"

static bool
CanIgnoreLpHeader(uint64_t tlvType)
{
  return 800 <= tlvType && tlvType <= 959 && (tlvType & 0x3) == 0x0;
}

NdnError
LpHeader_FromPacket(LpHeader* lph,
                    struct rte_mbuf* pkt,
                    uint32_t* payloadOff,
                    uint32_t* tlvSize)
{
  memset(lph, 0, sizeof(LpHeader));
  lph->l2.fragCount = 1;

  TlvDecodePos d0;
  MbufLoc_Init(&d0, pkt);
  TlvElement lppEle;
  NdnError e = DecodeTlvElement(&d0, &lppEle);
  RETURN_IF_ERROR;
  *tlvSize = lppEle.size;
  if (lppEle.type == TT_Interest || lppEle.type == TT_Data) {
    *payloadOff = 0;
    return NdnError_OK;
  }

  *payloadOff = lppEle.size;

  TlvDecodePos d1;
  TlvElement_MakeValueDecoder(&lppEle, &d1);
  TlvElement ele1;
  while ((e = DecodeTlvElement(&d1, &ele1)) == NdnError_OK) {
    switch (ele1.type) {
      case TT_LpPayload:
        *payloadOff = lppEle.size - ele1.length;
        goto FOUND_PAYLOAD;
      case TT_LpSeqNo:
        // NDNLPv2 spec defines SeqNo as "fixed-width unsigned integer",
        // but ndn-cxx implements it as nonNegativeInteger.
        // https://redmine.named-data.net/issues/4403
        TlvElement_ReadNonNegativeInteger(&ele1, &lph->l2.seqNum);
        break;
      case TT_FragIndex: {
        uint64_t v;
        TlvElement_ReadNonNegativeInteger(&ele1, &v);
        if (v > UINT16_MAX) {
          return NdnError_LengthOverflow;
        }
        lph->l2.fragIndex = v;
        break;
      }
      case TT_FragCount: {
        uint64_t v;
        TlvElement_ReadNonNegativeInteger(&ele1, &v);
        if (v > UINT16_MAX) {
          return NdnError_LengthOverflow;
        }
        lph->l2.fragCount = v;
        break;
      }
      case TT_PitToken: {
        if (unlikely(ele1.length != 8)) {
          return NdnError_BadPitToken;
        }
        TlvDecodePos d2;
        TlvElement_MakeValueDecoder(&ele1, &d2);
        rte_le64_t v;
        MbufLoc_ReadU64(&d2, &v);
        lph->l3.pitToken = rte_le_to_cpu_64(v);
        break;
      }
      case TT_Nack: {
        TlvDecodePos d2;
        TlvElement_MakeValueDecoder(&ele1, &d2);
        TlvElement ele2;
        if (likely(DecodeTlvElementExpectType(&d2, TT_NackReason, &ele2) ==
                   NdnError_OK)) {
          uint64_t v;
          TlvElement_ReadNonNegativeInteger(&ele2, &v);
          if (v > UINT8_MAX) {
            return NdnError_LengthOverflow;
          }
          lph->l3.nackReason = v;
        } else {
          lph->l3.nackReason = NackReason_Unspecified;
        }
        break;
      }
      case TT_CongestionMark: {
        uint64_t v;
        TlvElement_ReadNonNegativeInteger(&ele1, &v);
        if (v > UINT8_MAX) {
          return NdnError_LengthOverflow;
        }
        lph->l3.congMark = v;
        break;
      }
      default:
        if (!CanIgnoreLpHeader(ele1.type)) {
          return NdnError_UnknownCriticalLpHeader;
        }
        break;
    }
  }

FOUND_PAYLOAD:;
  if (unlikely(!MbufLoc_IsEnd(&d1))) {
    return NdnError_LpHasTrailer;
  }
  if (unlikely(lph->l2.fragIndex >= lph->l2.fragCount)) {
    return NdnError_FragIndexExceedFragCount;
  }
  return NdnError_OK;
}

void
PrependLpHeader(struct rte_mbuf* m, const LpHeader* lph, uint32_t payloadL)
{
  assert(rte_pktmbuf_headroom(m) >= PrependLpHeader_GetHeadroom());
  TlvEncoder* en = MakeTlvEncoder_Unchecked(m);

  uint16_t size0 = m->data_len;
  if (likely(payloadL) != 0) {
    PrependVarNum(en, payloadL);
    PrependVarNum(en, TT_LpPayload);
  }
  uint16_t size1 = m->data_len;

  if (lph->l2.fragIndex == 0) {
    if (lph->l3.congMark != 0) {
      typedef struct CongMarkF
      {
        uint8_t congMarkT[3];
        uint8_t congMarkL;
        uint8_t congMarkV;
      } __rte_packed CongMarkF;

      CongMarkF* f = (CongMarkF*)TlvEncoder_Prepend(en, sizeof(CongMarkF));
      assert(SizeofVarNum(TT_CongestionMark) == sizeof(f->congMarkT));
      EncodeVarNum(f->congMarkT, TT_CongestionMark);
      f->congMarkL = 1;
      f->congMarkV = lph->l3.congMark;
    }

    if (lph->l3.nackReason != NackReason_None) {
      if (unlikely(lph->l3.nackReason == NackReason_Unspecified)) {
        PrependVarNum(en, 0);
        PrependVarNum(en, TT_Nack);
      } else {
        typedef struct NackF
        {
          uint8_t nackT[3];
          uint8_t nackL;
          uint8_t nackReasonT[3];
          uint8_t nackReasonL;
          uint8_t nackReasonV;
        } __rte_packed NackF;

        NackF* f = (NackF*)TlvEncoder_Prepend(en, sizeof(NackF));
        assert(SizeofVarNum(TT_Nack) == sizeof(f->nackT));
        EncodeVarNum(f->nackT, TT_Nack);
        f->nackL = 5;
        assert(SizeofVarNum(TT_NackReason) == sizeof(f->nackReasonT));
        EncodeVarNum(f->nackReasonT, TT_NackReason);
        f->nackReasonL = 1;
        f->nackReasonV = lph->l3.nackReason;
      }
    }

    if (lph->l3.pitToken != 0) {
      typedef struct PitTokenF
      {
        uint8_t pitTokenT;
        uint8_t pitTokenL;
        rte_le64_t pitTokenV;
      } __rte_packed PitTokenF;

      PitTokenF* f = (PitTokenF*)TlvEncoder_Prepend(en, sizeof(PitTokenF));
      f->pitTokenT = TT_PitToken;
      f->pitTokenL = 8;
      *(unaligned_uint64_t*)&f->pitTokenV = rte_cpu_to_le_64(lph->l3.pitToken);
    }
  }

  if (lph->l2.fragCount > 1) {
    typedef struct FragF
    {
      uint8_t seqNumT;
      uint8_t seqNumL;
      rte_be64_t seqNumV;

      // FragIndex and FragCount are NonNegativeInteger fields, but NDN protocol does not
      // require NonNegativeInteger to use minimal length encoding.
      uint8_t fragIndexT;
      uint8_t fragIndexL;
      rte_be16_t fragIndexV;

      uint8_t fragCountT;
      uint8_t fragCountL;
      rte_be16_t fragCountV;
    } __rte_packed FragF;

    FragF* f = (FragF*)TlvEncoder_Prepend(en, sizeof(FragF));
    f->seqNumT = TT_LpSeqNo;
    f->seqNumL = 8;
    *(unaligned_uint64_t*)&f->seqNumV = rte_cpu_to_be_64(lph->l2.seqNum);
    f->fragIndexT = TT_FragIndex;
    f->fragIndexL = 2;
    *(unaligned_uint16_t*)&f->fragIndexV = rte_cpu_to_be_16(lph->l2.fragIndex);
    f->fragCountT = TT_FragCount;
    f->fragCountL = 2;
    *(unaligned_uint16_t*)&f->fragCountV = rte_cpu_to_be_16(lph->l2.fragCount);
  }

  if (m->data_len == size1) { // no LP headers
    rte_pktmbuf_adj(m, size1 - size0);
    return;
  }

  PrependVarNum(en, m->data_len - size0 + payloadL);
  PrependVarNum(en, TT_LpPacket);
}
