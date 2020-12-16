#include "interest.h"
#include "packet.h"
#include "tlv-decoder.h"
#include "tlv-encoder.h"
#include <rte_random.h>

void
NonceGen_Init(NonceGen* g)
{
  pcg32_srandom_r(&g->rng, rte_rand(), rte_rand());
}

static __rte_always_inline bool
PInterest_ParseDelegation_(PInterest* interest, TlvDecoder* d)
{
  TlvDecoder_EachTL (d, type, length) {
    switch (type) {
      case TtPreference:
        TlvDecoder_Skip(d, length);
        break;
      case TtName: {
        const uint8_t* v;
        if (unlikely(length > NameMaxLength || (v = TlvDecoder_Linearize(d, length)) == NULL)) {
          return false;
        }
        interest->fwHintL[interest->nFwHints] = length;
        interest->fwHintV[interest->nFwHints] = v;
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
  ++interest->nFwHints;
  return likely(d->length == 0);
}

static bool
PInterest_ParseFwHint_(PInterest* interest, TlvDecoder* d)
{
  TlvDecoder_EachTL (d, type, length) {
    switch (type) {
      case TtDelegation: {
        if (unlikely(interest->nFwHints >= PInterestMaxFwHints)) {
          TlvDecoder_Skip(d, length);
          break;
        }

        TlvDecoder vd;
        TlvDecoder_MakeValueDecoder(d, length, &vd);
        if (unlikely(!PInterest_ParseDelegation_(interest, &vd))) {
          return false;
        }
        d->m = vd.m; // mbuf may change when linearizing
        d->offset = vd.offset;
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
  return likely(d->length == 0);
}

bool
PInterest_Parse(PInterest* interest, struct rte_mbuf* pkt)
{
  NDNDPDK_ASSERT(RTE_MBUF_DIRECT(pkt) && rte_mbuf_refcnt_read(pkt) == 1);
  *interest = (const PInterest){ 0 };
  interest->lifetime = DefaultInterestLifetime;
  interest->hopLimit = UINT8_MAX;
  interest->activeFwHint = -1;

  TlvDecoder d;
  TlvDecoder_Init(&d, pkt);
  uint32_t length0, type0 = TlvDecoder_ReadTL(&d, &length0);
  NDNDPDK_ASSERT(type0 == TtInterest);

  uint32_t posStart = d.length, posNonce = 0, posEndGuider = 0;
  TlvDecoder_EachTL (&d, type, length) {
    switch (type) {
      case TtName: {
        const uint8_t* v;
        if (unlikely(length > NameMaxLength || (v = TlvDecoder_Linearize(&d, length)) == NULL)) {
          return false;
        }
        LName lname = LName_Init(length, v);
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
        TlvDecoder vd;
        TlvDecoder_MakeValueDecoder(&d, length, &vd);
        if (unlikely(!PInterest_ParseFwHint_(interest, &vd))) {
          return false;
        }
        d.m = vd.m; // mbuf may change when linearizing
        d.offset = vd.offset;
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
  bool ok = PName_Parse(&interest->fwHint, LName_Init(interest->fwHintL[i], interest->fwHintV[i]));
  interest->activeFwHint = likely(ok) ? i : -1;
  return -1;
}

typedef struct GuiderFields
{
  uint8_t nonceT;
  uint8_t nonceL;
  unaligned_uint32_t nonceV;

  uint8_t lifetimeT;
  uint8_t lifetimeL;
  unaligned_uint32_t lifetimeV;

  uint8_t hopLimitT;
  uint8_t hopLimitL;
  uint8_t hopLimitV;
} __rte_packed GuiderFields;

__attribute__((nonnull)) static void
ModifyGuiders_Append(struct rte_mbuf* m, InterestGuiders g)
{
  GuiderFields* f = (GuiderFields*)rte_pktmbuf_append(m, sizeof(GuiderFields));
  NDNDPDK_ASSERT(f != NULL);
  f->nonceT = TtNonce;
  f->nonceL = sizeof(f->nonceV);
  f->nonceV = rte_cpu_to_be_32(g.nonce);
  f->lifetimeT = TtInterestLifetime;
  f->lifetimeL = sizeof(f->lifetimeV);
  f->lifetimeV = rte_cpu_to_be_32(g.lifetime);
  f->hopLimitT = TtHopLimit;
  f->hopLimitL = sizeof(f->hopLimitV);
  f->hopLimitV = g.hopLimit;
}

__attribute__((nonnull, returns_nonnull)) static Packet*
ModifyGuiders_Finish(struct rte_mbuf* m)
{
  TlvEncoder_PrependTL(m, TtInterest, m->pkt_len);
  Packet* output = Packet_FromMbuf(m);
  Packet_SetType(output, PktSInterest);
  *Packet_GetLpL3Hdr(output) = (const LpL3){ 0 };
  return output;
}

__attribute__((nonnull)) static Packet*
ModifyGuiders_Linear(Packet* npkt, InterestGuiders guiders, PacketMempools* mp,
                     uint16_t fragmentPayloadSize)
{
  PInterest* interest = Packet_GetInterestHdr(npkt);
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  TlvDecoder d;
  TlvDecoder_Init(&d, pkt);
  uint32_t length0, type0 = TlvDecoder_ReadTL(&d, &length0);
  NDNDPDK_ASSERT(type0 == TtInterest);

  fragmentPayloadSize -= sizeof(GuiderFields); // keep room for guiders in any fragment
  uint32_t fragCount = DIV_CEIL(d.length - interest->guiderSize, fragmentPayloadSize);
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
  return ModifyGuiders_Finish(frames[0]);
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
  TlvDecoder d;
  TlvDecoder_Init(&d, Packet_ToMbuf(npkt));
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
  return ModifyGuiders_Finish(segs[0]);
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
