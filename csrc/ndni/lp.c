#include "lp.h"
#include "../core/base16.h"
#include "tlv-decoder.h"
#include "tlv-encoder.h"

static RTE_DEFINE_PER_LCORE(
  struct { char buffer[Base16_BufferSize(RTE_SIZEOF_FIELD(LpPitToken, value))]; },
  LpPitTokenStringBuffer);

const char*
LpPitToken_ToString(const LpPitToken* token)
{
  if (unlikely(token->length == 0)) {
    return "(empty)";
  }
#define buffer (RTE_PER_LCORE(LpPitTokenStringBuffer).buffer)
  Base16_Encode(buffer, sizeof(buffer), token->value, token->length);
  return buffer;
#undef buffer
}

static __rte_always_inline bool
LpHeader_IsCriticalType(uint32_t type)
{
  return type < 800 || type > 959 || (type & 0x03) != 0x00;
}

__attribute__((nonnull)) static __rte_always_inline bool
LpHeader_ParseNack(LpHeader* lph, TlvDecoder* d)
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
        if (LpHeader_IsCriticalType(type)) {
          return false;
        }
        TlvDecoder_Skip(d, length);
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

  TlvDecoder d = TlvDecoder_Init(pkt);
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

  uint64_t seqNum = 0;
  TlvDecoder_EachTL (&d, type, length) {
    switch (type) {
      case TtLpPayload: {
        pkt->data_off += d.offset;
        pkt->pkt_len = pkt->data_len = length;
        goto FOUND_PAYLOAD;
      }
      case TtLpSeqNum: {
        if (unlikely(length != 8 || !TlvDecoder_ReadNniTo(&d, length, &seqNum))) {
          return false;
        }
        break;
      }
      case TtFragIndex: {
        if (unlikely(!TlvDecoder_ReadNniTo(&d, length, LpMaxFragments - 1, &lph->l2.fragIndex))) {
          return false;
        }
        break;
      }
      case TtFragCount: {
        if (unlikely(!TlvDecoder_ReadNniTo(&d, length, LpMaxFragments, &lph->l2.fragCount))) {
          return false;
        }
        break;
      }
      case TtPitToken: {
        if (unlikely(length > sizeof(lph->l3.pitToken.value))) {
          return false;
        }
        lph->l3.pitToken.length = length;
        TlvDecoder_Copy(&d, lph->l3.pitToken.value, length);
        break;
      }
      case TtNack: {
        TlvDecoder vd = TlvDecoder_MakeValueDecoder(&d, length);
        if (unlikely(!LpHeader_ParseNack(lph, &vd))) {
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
        if (LpHeader_IsCriticalType(type)) {
          return false;
        }
        TlvDecoder_Skip(&d, length);
        break;
    }
  }

  pkt->pkt_len = pkt->data_len = 0; // no payload
FOUND_PAYLOAD:;
  lph->l2.seqNumBase = seqNum - lph->l2.fragIndex;
  return lph->l2.fragIndex < lph->l2.fragCount;
}

void
LpHeader_Prepend(struct rte_mbuf* pkt, const LpL3* l3, const LpL2* l2)
{
  NDNDPDK_ASSERT(rte_pktmbuf_headroom(pkt) >= LpHeaderHeadroom);
  TlvEncoder_PrependTL(pkt, TtLpPayload, pkt->pkt_len);

  if (likely(l2->fragIndex == 0)) {
    if (unlikely(l3->congMark != 0)) {
      typedef struct CongMarkF
      {
        unaligned_uint32_t congMarkTL;
        uint8_t congMarkV;
      } __rte_packed CongMarkF;

      CongMarkF* f = (CongMarkF*)rte_pktmbuf_prepend(pkt, sizeof(CongMarkF));
      f->congMarkTL = TlvEncoder_ConstTL3(TtCongestionMark, sizeof(f->congMarkV));
      f->congMarkV = l3->congMark;
    }

    if (unlikely(l3->nackReason != NackNone)) {
      if (unlikely(l3->nackReason == NackUnspecified)) {
        TlvEncoder_PrependTL(pkt, TtNack, 0);
      } else {
        typedef struct NackF
        {
          unaligned_uint32_t nackTL;
          unaligned_uint32_t nackReasonTL;
          uint8_t nackReasonV;
        } __rte_packed NackF;

        NackF* f = (NackF*)rte_pktmbuf_prepend(pkt, sizeof(NackF));
        f->nackTL = TlvEncoder_ConstTL3(TtNack, sizeof(f->nackReasonTL) + sizeof(f->nackReasonV));
        f->nackReasonTL = TlvEncoder_ConstTL3(TtNackReason, sizeof(f->nackReasonV));
        f->nackReasonV = l3->nackReason;
      }
    }

    if (likely(l3->pitToken.length > 0)) {
      typedef struct PitTokenF
      {
        uint8_t pitTokenT;
        uint8_t pitTokenLV[];
      } __rte_packed PitTokenF;

      size_t sizeofLV = sizeof(l3->pitToken.length) + l3->pitToken.length;
      PitTokenF* f = (PitTokenF*)rte_pktmbuf_prepend(pkt, sizeof(f->pitTokenT) + sizeofLV);
      f->pitTokenT = TtPitToken;
      rte_memcpy(f->pitTokenLV, &l3->pitToken, sizeofLV);
    }
  }

  if (unlikely(l2->fragCount > 1)) {
    NDNDPDK_ASSERT(l2->fragIndex < l2->fragCount);
    NDNDPDK_ASSERT(l2->fragCount <= LpMaxFragments);

    typedef struct FragF
    {
      unaligned_uint16_t seqNumTL;
      unaligned_uint64_t seqNumV;
      unaligned_uint16_t fragIndexTL;
      uint8_t fragIndexV;
      unaligned_uint16_t fragCountTL;
      uint8_t fragCountV;
    } __rte_packed FragF;

    FragF* f = (FragF*)rte_pktmbuf_prepend(pkt, sizeof(FragF));
    f->seqNumTL = TlvEncoder_ConstTL1(TtLpSeqNum, sizeof(f->seqNumV));
    f->seqNumV = rte_cpu_to_be_64(LpL2_GetSeqNum(l2));
    f->fragIndexTL = TlvEncoder_ConstTL1(TtFragIndex, sizeof(f->fragIndexV));
    f->fragIndexV = l2->fragIndex;
    f->fragCountTL = TlvEncoder_ConstTL1(TtFragCount, sizeof(f->fragCountV));
    f->fragCountV = l2->fragCount;
  }

  TlvEncoder_PrependTL(pkt, TtLpPacket, pkt->pkt_len);
}
