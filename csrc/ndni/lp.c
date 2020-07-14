#include "lp.h"
#include "tlv-decoder.h"
#include "tlv-encoder.h"

static __rte_always_inline bool
LpHeader_IsCriticalType_(uint32_t type)
{
  return type < 800 || type > 959 || (type & 0x03) != 0x00;
}

static __rte_always_inline bool
LpHeader_ParseNack_(LpHeader* lph, TlvDecoder* d)
{
  lph->l3.nackReason = NackUnspecified;
  TlvDecoder_EachTL (d, type, length) {
    switch (type) {
      case TtNackReason:
        if (unlikely(!TlvDecoder_ReadNniTo(d, length, &lph->l3.nackReason))) {
          lph->l3.nackReason = NackUnspecified;
        }
        break;
      default:
        if (LpHeader_IsCriticalType_(type)) {
          return false;
        }
        break;
    }
  }
  return true;
}

bool
LpHeader_Parse(LpHeader* lph, struct rte_mbuf* pkt)
{
  NDNDPDK_ASSERT(RTE_MBUF_DIRECT(pkt) && rte_pktmbuf_is_contiguous(pkt) &&
                 rte_mbuf_refcnt_read(pkt) == 1);
  *lph = (const LpHeader){ 0 };
  lph->l2.fragCount = 1;

  TlvDecoder d;
  TlvDecoder_New(&d, pkt);
  uint32_t length0, type0 = TlvDecoder_ReadTL(&d, &length0);
  pkt->pkt_len = pkt->data_len = d.offset + length0; // strip Ethernet trailer, if any
  d.length = length0;
  switch (type0) {
    case TtInterest:
    case TtData:
      return true;
    case TtLpPacket:
      break;
    default:
      return false;
  }

  TlvDecoder_EachTL (&d, type, length) {
    switch (type) {
      case TtLpPayload: {
        pkt->data_off += d.offset;
        pkt->pkt_len = pkt->data_len = length;
        goto FOUND_PAYLOAD;
      }
      case TtLpSeqNum: {
        if (unlikely(length != 8 || !TlvDecoder_ReadNniTo(&d, length, &lph->l2.seqNum))) {
          return false;
        }
        break;
      }
      case TtFragIndex: {
        if (unlikely(!TlvDecoder_ReadNniTo(&d, length, &lph->l2.fragIndex))) {
          return false;
        }
        break;
      }
      case TtFragCount: {
        if (unlikely(!TlvDecoder_ReadNniTo(&d, length, &lph->l2.fragCount))) {
          return false;
        }
        break;
      }
      case TtPitToken: {
        if (unlikely(length != 8 || !TlvDecoder_ReadNniTo(&d, length, &lph->l3.pitToken))) {
          return false;
        }
        break;
      }
      case TtNack: {
        TlvDecoder vd;
        TlvDecoder_MakeValueDecoder(&d, length, &vd);
        if (unlikely(!LpHeader_ParseNack_(lph, &vd))) {
          return false;
        }
        break;
      }
      case TtCongestionMark: {
        if (unlikely(!TlvDecoder_ReadNniTo(&d, length, &lph->l3.congMark))) {
          return false;
        }
        break;
      }
      default:
        if (LpHeader_IsCriticalType_(type)) {
          return false;
        }
        TlvDecoder_Skip(&d, length);
        break;
    }
  }

  pkt->pkt_len = pkt->data_len = 0; // no payload
FOUND_PAYLOAD:;
  return lph->l2.fragIndex < lph->l2.fragCount;
}

void
LpHeader_Prepend(struct rte_mbuf* pkt, const LpL3* l3, const LpL2* l2)
{
  NDNDPDK_ASSERT(rte_pktmbuf_headroom(pkt) >= LpHeaderEstimatedHeadroom);
  TlvEncoder_PrependTL(pkt, TtLpPayload, pkt->pkt_len);

  if (likely(l2->fragIndex == 0)) {
    if (unlikely(l3->congMark != 0)) {
      typedef struct CongMarkF
      {
        uint8_t congMarkT[3];
        uint8_t congMarkL;
        uint8_t congMarkV;
      } __rte_packed CongMarkF;

      CongMarkF* f = (CongMarkF*)rte_pktmbuf_prepend(pkt, sizeof(CongMarkF));
      NDNDPDK_ASSERT(TlvEncoder_SizeofVarNum(TtCongestionMark) == sizeof(f->congMarkT));
      TlvEncoder_WriteVarNum(f->congMarkT, TtCongestionMark);
      f->congMarkL = 1;
      f->congMarkV = l3->congMark;
    }

    if (unlikely(l3->nackReason != NackNone)) {
      if (unlikely(l3->nackReason == NackUnspecified)) {
        TlvEncoder_PrependTL(pkt, TtNack, 0);
      } else {
        typedef struct NackF
        {
          uint8_t nackT[3];
          uint8_t nackL;
          uint8_t nackReasonT[3];
          uint8_t nackReasonL;
          uint8_t nackReasonV;
        } __rte_packed NackF;

        NackF* f = (NackF*)rte_pktmbuf_prepend(pkt, sizeof(NackF));
        TlvEncoder_WriteVarNum(f->nackT, TtNack);
        f->nackL = 5;
        TlvEncoder_WriteVarNum(f->nackReasonT, TtNackReason);
        f->nackReasonL = 1;
        f->nackReasonV = l3->nackReason;
      }
    }

    if (likely(l3->pitToken != 0)) {
      typedef struct PitTokenF
      {
        uint8_t pitTokenT;
        uint8_t pitTokenL;
        unaligned_uint64_t pitTokenV;
      } __rte_packed PitTokenF;

      PitTokenF* f = (PitTokenF*)rte_pktmbuf_prepend(pkt, sizeof(PitTokenF));
      f->pitTokenT = TtPitToken;
      f->pitTokenL = 8;
      f->pitTokenV = rte_cpu_to_be_64(l3->pitToken);
    }
  }

  if (unlikely(l2->fragCount > 1)) {
    typedef struct FragF
    {
      uint8_t seqNumT;
      uint8_t seqNumL;
      unaligned_uint64_t seqNumV;

      // FragIndex and FragCount are NonNegativeInteger fields, but NDN protocol does not
      // require NonNegativeInteger to use minimal length encoding.
      uint8_t fragIndexT;
      uint8_t fragIndexL;
      unaligned_uint16_t fragIndexV;

      uint8_t fragCountT;
      uint8_t fragCountL;
      unaligned_uint16_t fragCountV;
    } __rte_packed FragF;

    FragF* f = (FragF*)rte_pktmbuf_prepend(pkt, sizeof(FragF));
    f->seqNumT = TtLpSeqNum;
    f->seqNumL = 8;
    f->seqNumV = rte_cpu_to_be_64(l2->seqNum);
    f->fragIndexT = TtFragIndex;
    f->fragIndexL = 2;
    f->fragIndexV = rte_cpu_to_be_16(l2->fragIndex);
    f->fragCountT = TtFragCount;
    f->fragCountL = 2;
    f->fragCountV = rte_cpu_to_be_16(l2->fragCount);
  }

  TlvEncoder_PrependTL(pkt, TtLpPacket, pkt->pkt_len);
}
