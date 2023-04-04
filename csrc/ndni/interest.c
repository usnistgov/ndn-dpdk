#include "interest.h"
#include "packet.h"
#include "tlv-decoder.h"
#include "tlv-encoder.h"

__attribute__((nonnull)) static inline bool
PInterest_ParseFwHint(PInterest* interest, TlvDecoder* d)
{
  TlvDecoder_EachTL (d, type, length) {
    switch (type) {
      case TtName: {
        if (unlikely(interest->nFwHints >= PInterestMaxFwHints)) {
          TlvDecoder_Skip(d, length);
          break;
        }

        const uint8_t* v = NULL;
        if (unlikely(length > NameMaxLength || (v = TlvDecoder_Linearize(d, length)) == NULL)) {
          return false;
        }
        interest->fwHintV[interest->nFwHints] = v;
        interest->fwHintL[interest->nFwHints] = length;
        ++interest->nFwHints;
        break;
      }
      default:
        if (TlvDecoder_IsCriticalType(type)) {
          return false;
        }
        TlvDecoder_Skip(d, length);
        break;
    }
  }
  return likely(d->length == 0 && interest->nFwHints > 0);
}

bool
PInterest_Parse(PInterest* interest, struct rte_mbuf* pkt, ParseFor parseFor)
{
  NDNDPDK_ASSERT(RTE_MBUF_DIRECT(pkt) && rte_mbuf_refcnt_read(pkt) == 1);
  *interest = (const PInterest){ 0 };
  interest->lifetime = DefaultInterestLifetime;
  interest->hopLimit = UINT8_MAX;
  interest->activeFwHint = -1;

  TlvDecoder d = TlvDecoder_Init(pkt);
  uint32_t length0, type0 = TlvDecoder_ReadTL(&d, &length0);
  NDNDPDK_ASSERT(type0 == TtInterest);

  uint32_t posStart = d.length, posNonce = 0, posEndGuider = 0;
  TlvDecoder_EachTL (&d, type, length) {
    switch (type) {
      case TtName: {
        LName lname = (LName){ .length = length };
        if (unlikely(length > NameMaxLength ||
                     (lname.value = TlvDecoder_Linearize(&d, length)) == NULL)) {
          return false;
        }
        if (unlikely(!PName_Parse(&interest->name, lname))) {
          return false;
        }
        break;
      }
      case TtCanBePrefix: {
        interest->canBePrefix = true;
        TlvDecoder_Skip(&d, length);
        break;
      }
      case TtMustBeFresh: {
        interest->mustBeFresh = true;
        TlvDecoder_Skip(&d, length);
        break;
      }
      case TtForwardingHint: {
        if (parseFor == ParseForApp) {
          TlvDecoder_Skip(&d, length);
        } else {
          TlvDecoder vd = TlvDecoder_MakeValueDecoder(&d, length);
          if (unlikely(!PInterest_ParseFwHint(interest, &vd))) {
            return false;
          }
          d.m = vd.m; // mbuf may change when linearizing
          d.offset = vd.offset;
        }
        break;
      }
      case TtNonce: {
        if (unlikely(length != 4 || !TlvDecoder_ReadNniTo(&d, length, &interest->nonce))) {
          return false;
        }
        posEndGuider = d.length;
        posNonce = d.length + 6;
        break;
      }
      case TtInterestLifetime: {
        if (unlikely(!TlvDecoder_ReadNniTo(&d, length, &interest->lifetime))) {
          return false;
        }
        posEndGuider = d.length;
        break;
      }
      case TtHopLimit: {
        if (unlikely(length != 1 || !TlvDecoder_ReadNniTo(&d, length, &interest->hopLimit))) {
          return false;
        }
        posEndGuider = d.length;
        break;
      }
      case TtAppParameters: {
        goto FOUND_PARAMETERS;
      }
      default:
        if (TlvDecoder_IsCriticalType(type)) {
          return false;
        }
        TlvDecoder_Skip(&d, length);
        break;
    }
  }

FOUND_PARAMETERS:;
  uint32_t guiderSize = posNonce - posEndGuider;
  if (unlikely(interest->name.nComps == 0) || // missing or empty name
      unlikely(posNonce == 0) ||              // missing Nonce
      unlikely(guiderSize > UINT8_MAX)        // too many unrecognized fields amid guiders
  ) {
    return false;
  }
  interest->nonceOffset = posStart - posNonce;
  interest->guiderSize = guiderSize;
  return true;
}

bool
PInterest_SelectFwHint(PInterest* interest, int i)
{
  NDNDPDK_ASSERT(i >= 0 && i < (int)interest->nFwHints);
  bool ok = PName_Parse(&interest->fwHint,
                        (LName){ .length = interest->fwHintL[i], .value = interest->fwHintV[i] });
  interest->activeFwHint = likely(ok) ? i : -1;
  return ok;
}

typedef struct GuiderFields
{
  unaligned_uint16_t nonceTL;
  unaligned_uint32_t nonceV;

  unaligned_uint16_t lifetimeTL;
  unaligned_uint32_t lifetimeV;

  unaligned_uint16_t hopLimitTL;
  uint8_t hopLimitV;
} __rte_packed GuiderFields;

__attribute__((nonnull)) static void
ModifyGuiders_Append(struct rte_mbuf* m, InterestGuiders g)
{
  GuiderFields* f = (GuiderFields*)rte_pktmbuf_append(m, sizeof(GuiderFields));
  NDNDPDK_ASSERT(f != NULL);
  f->nonceTL = TlvEncoder_ConstTL1(TtNonce, sizeof(f->nonceV));
  f->nonceV = rte_cpu_to_be_32(g.nonce);
  f->lifetimeTL = TlvEncoder_ConstTL1(TtInterestLifetime, sizeof(f->lifetimeV));
  f->lifetimeV = rte_cpu_to_be_32(g.lifetime);
  f->hopLimitTL = TlvEncoder_ConstTL1(TtHopLimit, sizeof(f->hopLimitV));
  f->hopLimitV = g.hopLimit;
}

__attribute__((nonnull)) static Packet*
ModifyGuiders_Linear(Packet* npkt, InterestGuiders guiders, PacketMempools* mp,
                     uint16_t fragmentPayloadSize)
{
  PInterest* interest = Packet_GetInterestHdr(npkt);
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  TlvDecoder d = TlvDecoder_Init(pkt);
  uint32_t length0, type0 = TlvDecoder_ReadTL(&d, &length0);
  NDNDPDK_ASSERT(type0 == TtInterest);

  fragmentPayloadSize -= sizeof(GuiderFields); // keep room for guiders in any fragment
  uint32_t fragCount = SPDK_CEIL_DIV(d.length - interest->guiderSize, fragmentPayloadSize);
  NDNDPDK_ASSERT(fragCount < LpMaxFragments);
  struct rte_mbuf* frames[LpMaxFragments];
  if (unlikely(rte_pktmbuf_alloc_bulk(mp->packet, frames, fragCount) != 0)) {
    return NULL;
  }

  uint32_t fragIndex = 0;
  frames[fragIndex]->data_off = RTE_PKTMBUF_HEADROOM + LpHeaderHeadroom + //
                                L3TypeLengthHeadroom;                     // Interest TL
  TlvDecoder_Fragment(&d, interest->nonceOffset, frames, &fragIndex, fragCount, fragmentPayloadSize,
                      RTE_PKTMBUF_HEADROOM + LpHeaderHeadroom);

  TlvDecoder_Skip(&d, interest->guiderSize);
  ModifyGuiders_Append(frames[fragIndex], guiders);

  TlvDecoder_Fragment(&d, d.length, frames, &fragIndex, fragCount, fragmentPayloadSize,
                      RTE_PKTMBUF_HEADROOM + LpHeaderHeadroom);

  Mbuf_ChainVector(frames, fragCount);
  return Packet_EncodeFinish_(frames[0], TtInterest, PktSInterest);
}

__attribute__((nonnull)) static Packet*
ModifyGuiders_Chained(Packet* npkt, InterestGuiders guiders, PacketMempools* mp)
{
  // segs[0] = Interest TL, with headroom for lower layer headers
  // segs[1] = clone of Interest V before Nonce, such as Name and ForwardingHint
  // segs[2] = new guiders
  // seg3    = (optional) clone of Interest V after guiders, such as AppParameters
  struct rte_mbuf* segs[3];
  if (unlikely(rte_pktmbuf_alloc_bulk(mp->header, segs, 2) != 0)) {
    return NULL;
  }
  segs[2] = segs[1];

  PInterest* interest = Packet_GetInterestHdr(npkt);
  TlvDecoder d = TlvDecoder_Init(Packet_ToMbuf(npkt));
  uint32_t length0, type0 = TlvDecoder_ReadTL(&d, &length0);
  NDNDPDK_ASSERT(type0 == TtInterest);

  struct rte_mbuf* last1 = NULL;
  segs[1] = TlvDecoder_Clone(&d, interest->nonceOffset, mp->indirect, &last1);
  if (unlikely(segs[1] == NULL)) {
    rte_pktmbuf_free_bulk(segs, RTE_DIM(segs));
    return NULL;
  }
  TlvDecoder_Skip(&d, interest->guiderSize);

  segs[2]->data_off = 0;
  ModifyGuiders_Append(segs[2], guiders);

  if (unlikely(d.length > 0)) {
    struct rte_mbuf* seg3 = TlvDecoder_Clone(&d, d.length, mp->indirect, NULL);
    if (unlikely(seg3 == NULL) || unlikely(!Mbuf_Chain(segs[2], segs[2], seg3))) {
      rte_pktmbuf_free_bulk(segs, RTE_DIM(segs));
      return NULL;
    }
  }

  if (unlikely(!Mbuf_Chain(segs[1], last1, segs[2]))) {
    rte_pktmbuf_free_bulk(segs, RTE_DIM(segs));
    return NULL;
  }

  segs[0]->data_off = RTE_PKTMBUF_HEADROOM + LpHeaderHeadroom + L3TypeLengthHeadroom;
  if (unlikely(!Mbuf_Chain(segs[0], segs[0], segs[1]))) {
    rte_pktmbuf_free_bulk(segs, 2);
    return NULL;
  }
  return Packet_EncodeFinish_(segs[0], TtInterest, PktSInterest);
}

Packet*
Interest_ModifyGuiders(Packet* npkt, InterestGuiders guiders, PacketMempools* mp,
                       PacketTxAlign align)
{
  if (align.linearize) {
    return ModifyGuiders_Linear(npkt, guiders, mp, align.fragmentPayloadSize);
  }
  return ModifyGuiders_Chained(npkt, guiders, mp);
}

Packet*
InterestTemplate_Encode(const InterestTemplate* tpl, struct rte_mbuf* m, LName suffix,
                        uint32_t nonce)
{
  NDNDPDK_ASSERT(RTE_MBUF_DIRECT(m) && rte_pktmbuf_is_contiguous(m) &&
                 rte_mbuf_refcnt_read(m) == 1 && m->data_len == 0 &&
                 m->buf_len >= InterestTemplateDataroom);
  m->data_off = m->buf_len;

  rte_memcpy(rte_pktmbuf_prepend(m, tpl->midLen), tpl->midBuf, tpl->midLen);
  unaligned_uint32_t* nonceV = rte_pktmbuf_mtod_offset(m, unaligned_uint32_t*, tpl->nonceVOffset);
  *nonceV = rte_cpu_to_be_32(nonce);

  rte_memcpy(rte_pktmbuf_prepend(m, suffix.length), suffix.value, suffix.length);
  rte_memcpy(rte_pktmbuf_prepend(m, tpl->prefixL), tpl->prefixV, tpl->prefixL);
  TlvEncoder_PrependTL(m, TtName, tpl->prefixL + suffix.length);

  TlvEncoder_PrependTL(m, TtInterest, m->pkt_len);

  Packet* output = Packet_FromMbuf(m);
  Packet_SetType(output, PktSInterest);
  *Packet_GetLpL3Hdr(output) = (const LpL3){ 0 };
  return output;
}
