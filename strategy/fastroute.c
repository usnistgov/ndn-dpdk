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

inline uint64_t
RxInterest(SgCtx* ctx)
{
  FibEntryInfo* fei = SgCtx_FibScratchT(ctx, FibEntryInfo);

  // TODO probe

  // unicast to selected nexthop
  // XXX this might incorrectly use a rejected nexthop
  if (fei->hasSelectedNexthop) {
    FaceId nh = ctx->fibEntry->nexthops[fei->selectedNexthop];
    SgForwardInterestResult res = SgForwardInterest(ctx, nh);
    if (res == SGFWDI_OK) {
      return 0;
    }
    // if unicasting fails, proceed to multicast
  }

  // multicast Interest
  SgFibNexthopIt it;
  for (SgFibNexthopIt_Init2(&it, ctx); SgFibNexthopIt_Valid(&it);
       SgFibNexthopIt_Next(&it)) {
    SgForwardInterest(ctx, it.nh);

    fei->hasSelectedNexthop = true;
    fei->selectedNexthop = it.i;
    // XXX RxData is not yet implemented, so this code selects the last nexthop
    // as a test.
  }
  return 4;
}

uint64_t
SgMain(SgCtx* ctx)
{
  switch (ctx->eventKind) {
    case SGEVT_INTEREST:
      return RxInterest(ctx);
    default:
      return 2;
  }
}
