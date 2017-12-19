#include "lp-pkt.h"
#include "nack-pkt.h"

static inline bool
CanIgnoreLpHeader(uint64_t tlvType)
{
  return 800 <= tlvType && tlvType <= 959 && (tlvType & 0x3) == 0x0;
}

NdnError
DecodeLpPkt(TlvDecoder* d, LpPkt* lpp)
{
  TlvElement lppEle;
  NdnError e = DecodeTlvElementExpectType(d, TT_LpPacket, &lppEle);
  RETURN_IF_UNLIKELY_ERROR;

  memset(lpp, 0, sizeof(LpPkt));
  lpp->fragCount = 1;

  TlvDecoder d1;
  TlvElement_MakeValueDecoder(&lppEle, &d1);
  TlvElement hdrEle;
  while ((e = DecodeTlvElement(&d1, &hdrEle)) == NdnError_OK) {
    switch (hdrEle.type) {
      case TT_LpPayload:
        TlvElement_MakeValueDecoder(&hdrEle, &lpp->payload);
        break;
      case TT_LpSeqNo:
        // NDNLPv2 spec defines SeqNo as "fixed-width unsigned integer",
        // but ndn-cxx implements it as nonNegativeInteger.
        TlvElement_ReadNonNegativeInteger(&hdrEle, &lpp->seqNo);
        break;
      case TT_FragIndex: {
        uint64_t v;
        TlvElement_ReadNonNegativeInteger(&hdrEle, &v);
        if (v > UINT16_MAX) {
          return NdnError_LengthOverflow;
        }
        lpp->fragIndex = v;
        break;
      }
      case TT_FragCount: {
        uint64_t v;
        TlvElement_ReadNonNegativeInteger(&hdrEle, &v);
        if (v > UINT16_MAX) {
          return NdnError_LengthOverflow;
        }
        lpp->fragCount = v;
        break;
      }
      case TT_Nack: {
        TlvDecoder d2;
        TlvElement_MakeValueDecoder(&hdrEle, &d2);
        TlvElement nackReasonEle;
        NdnError e2 =
          DecodeTlvElementExpectType(&d2, TT_NackReason, &nackReasonEle);
        if (unlikely(e2 == NdnError_Incomplete || e2 == NdnError_BadType)) {
          lpp->nackReason = NackReason_Unspecified;
          break;
        } else if (unlikely(e2 != NdnError_OK)) {
          return e2;
        }

        uint64_t v;
        TlvElement_ReadNonNegativeInteger(&nackReasonEle, &v);
        if (v > UINT8_MAX) {
          return NdnError_LengthOverflow;
        }
        lpp->nackReason = v;
        break;
      }
      case TT_CongestionMark: {
        uint64_t v;
        TlvElement_ReadNonNegativeInteger(&hdrEle, &v);
        if (v > UINT8_MAX) {
          return NdnError_LengthOverflow;
        }
        lpp->congMark = v;
        break;
      }
      default:
        if (!CanIgnoreLpHeader(hdrEle.type)) {
          return NdnError_UnknownCriticalLpHeader;
        }
        break;
    }
  }

  if (unlikely(!MbufLoc_IsEnd(&d1))) {
    return e;
  }
  if (unlikely(lpp->fragIndex >= lpp->fragCount)) {
    return NdnError_FragIndexExceedFragCount;
  }
  return NdnError_OK;
}