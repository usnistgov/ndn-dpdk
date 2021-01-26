/**
 * @file
 * The fast route strategy multicasts the first Interest, observes which
 * nexthop replies first, and keeps using it. It then periodically probes
 * an unselected nexthop, and switches to it if it is faster.
 */
#include "api.h"

// how often to send probe Interest, in number of packets
#define PROBE_INTERVAL 1024

enum StatusCode
{
  S_OK = 0,
  S_UNKNOWN = 2,
  S_UNICAST = 11,
  S_MULTICAST = 12,
  S_PROBE_OK = 21,
  S_PROBE_ERR = 22,
  S_PROBE_NONE = 23,
  S_NH_SAME = 50,
  S_NH_CHANGE = 51,
  S_NH_IGNORE = 52,
  S_NH_ERR = 53,
};

typedef struct FibEntryInfo
{
  uint16_t nUnicast;
  bool hasSelectedNexthop;
  uint8_t selectedNexthop;
} FibEntryInfo;

typedef struct PitEntryInfo
{
  bool multicastOrProbe;
} PitEntryInfo;

SUBROUTINE bool
Unicast(SgCtx* ctx)
{
  FibEntryInfo* fei = SgCtx_FibScratchT(ctx, FibEntryInfo);
  FaceID nh = ctx->fibEntry->nexthops[fei->selectedNexthop];
  SgForwardInterestResult res = SgForwardInterest(ctx, nh);
  return res == SGFWDI_OK;
}

SUBROUTINE uint64_t
Probe(SgCtx* ctx)
{
  FibEntryInfo* fei = SgCtx_FibScratchT(ctx, FibEntryInfo);
  PitEntryInfo* pei = SgCtx_PitScratchT(ctx, PitEntryInfo);

  if (ctx->fibEntry->nNexthops == 1) {
    return S_PROBE_NONE;
  }
  uint8_t i = ((uint16_t)ctx->now ^ (uint16_t)(ctx->now >> 16) ^ (uint16_t)(ctx->now >> 32) ^
               (uint16_t)(ctx->now >> 48)) %
              (uint16_t)(ctx->fibEntry->nNexthops - 1);
  if (i >= fei->selectedNexthop) {
    ++i;
  }

  FaceID nh = ctx->fibEntry->nexthops[i];
  SgForwardInterestResult res = SgForwardInterest(ctx, nh);
  if (res != SGFWDI_OK) {
    return S_PROBE_ERR;
  }
  pei->multicastOrProbe = true;
  return S_PROBE_OK;
}

SUBROUTINE uint64_t
RxInterest(SgCtx* ctx)
{
  FibEntryInfo* fei = SgCtx_FibScratchT(ctx, FibEntryInfo);
  PitEntryInfo* pei = SgCtx_PitScratchT(ctx, PitEntryInfo);

  // unicast to selected nexthop
  if (fei->hasSelectedNexthop && Unicast(ctx)) {
    if (++fei->nUnicast >= PROBE_INTERVAL) {
      fei->nUnicast = 0;
      return Probe(ctx);
    }
    return S_UNICAST;
  }
  // if unicasting fails, proceed to multicast

  // multicast Interest
  SgFibNexthopIt it;
  for (SgFibNexthopIt_Init2(&it, ctx); SgFibNexthopIt_Valid(&it); SgFibNexthopIt_Next(&it)) {
    SgForwardInterest(ctx, it.nh);
  }
  pei->multicastOrProbe = true;
  return S_MULTICAST;
}

SUBROUTINE uint64_t
RxData(SgCtx* ctx)
{
  FibEntryInfo* fei = SgCtx_FibScratchT(ctx, FibEntryInfo);
  PitEntryInfo* pei = SgCtx_PitScratchT(ctx, PitEntryInfo);

  if (fei->hasSelectedNexthop &&
      ctx->fibEntry->nexthops[fei->selectedNexthop] == ctx->pkt->rxFace) {
    return S_NH_SAME;
  }

  if (!fei->hasSelectedNexthop || pei->multicastOrProbe) {
    SgFibNexthopIt it;
    for (SgFibNexthopIt_Init(&it, ctx->fibEntry, 0); SgFibNexthopIt_Valid(&it);
         SgFibNexthopIt_Next(&it)) {
      if (it.nh == ctx->pkt->rxFace) {
        fei->hasSelectedNexthop = true;
        fei->selectedNexthop = it.i;
        return S_NH_CHANGE;
      }
    }
    return S_NH_ERR;
  }

  return S_NH_IGNORE;
}

SUBROUTINE uint64_t
RxNack(SgCtx* ctx)
{
  FibEntryInfo* fei = SgCtx_FibScratchT(ctx, FibEntryInfo);
  if (fei->hasSelectedNexthop &&
      ctx->fibEntry->nexthops[fei->selectedNexthop] == ctx->pkt->rxFace) {
    SgFibNexthopIt it;
    for (SgFibNexthopIt_Init2(&it, ctx); SgFibNexthopIt_Valid(&it); SgFibNexthopIt_Next(&it)) {
      if (it.i != fei->selectedNexthop) {
        SgForwardInterest(ctx, it.nh);
      }
    }
    fei->hasSelectedNexthop = false;
    fei->nUnicast = 0;
    return S_MULTICAST;
  }
  return S_OK;
}

uint64_t
SgMain(SgCtx* ctx)
{
  switch (ctx->eventKind) {
    case SGEVT_INTEREST:
      return RxInterest(ctx);
    case SGEVT_DATA:
      return RxData(ctx);
    case SGEVT_NACK:
      return RxNack(ctx);
    default:
      return S_UNKNOWN;
  }
}
