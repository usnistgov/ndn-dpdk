/** \file
 *  The fast route strategy multicasts the first Interest, observes which
 *  nexthop replies first, and keeps using it. It then periodically probes
 *  an unselected nexthop, and switches to it if it is faster.
 */
#include "api.h"

typedef struct FibEntryInfo
{
  bool hasSelectedNexthop;
  uint8_t selectedNexthop;
} FibEntryInfo;

SUBROUTINE uint64_t
RxInterest(SgCtx* ctx)
{
  FibEntryInfo* fei = SgCtx_FibScratchT(ctx, FibEntryInfo);

  // TODO probe

  // unicast to selected nexthop
  if (fei->hasSelectedNexthop) {
    SgFibNexthopIt it;
    for (SgFibNexthopIt_Init2(&it, ctx); SgFibNexthopIt_Valid(&it);
         SgFibNexthopIt_Next(&it)) {
      if (it.i == fei->selectedNexthop) {
        SgForwardInterestResult res = SgForwardInterest(ctx, it.nh);
        if (res == SGFWDI_OK) {
          return 0;
        }
        break;
      }
    }
    // if unicasting fails, proceed to multicast
  }

  // multicast Interest
  SgFibNexthopIt it;
  for (SgFibNexthopIt_Init2(&it, ctx); SgFibNexthopIt_Valid(&it);
       SgFibNexthopIt_Next(&it)) {
    SgForwardInterest(ctx, it.nh);
  }
  return 4;
}

SUBROUTINE uint64_t
RxData(SgCtx* ctx)
{
  FibEntryInfo* fei = SgCtx_FibScratchT(ctx, FibEntryInfo);

  SgFibNexthopIt it;
  for (SgFibNexthopIt_Init(&it, ctx->fibEntry, 0); SgFibNexthopIt_Valid(&it);
       SgFibNexthopIt_Next(&it)) {
    if (it.nh == ctx->pkt->rxFace) {
      fei->hasSelectedNexthop = true;
      fei->selectedNexthop = it.i;
      return 0;
    }
  }
  return 5;
}

SUBROUTINE uint64_t
RxNack(SgCtx* ctx)
{
  FibEntryInfo* fei = SgCtx_FibScratchT(ctx, FibEntryInfo);
  if (fei->hasSelectedNexthop &&
      ctx->fibEntry->nexthops[fei->selectedNexthop] == ctx->pkt->rxFace) {
    SgFibNexthopIt it;
    for (SgFibNexthopIt_Init2(&it, ctx); SgFibNexthopIt_Valid(&it);
         SgFibNexthopIt_Next(&it)) {
      if (it.i != fei->selectedNexthop) {
        SgForwardInterest(ctx, it.nh);
      }
    }
    fei->hasSelectedNexthop = false;
    return 0;
  }
  return 6;
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
      return 2;
  }
}
