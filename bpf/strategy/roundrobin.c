/**
 * @file
 * The roundrobin strategy uses each nexthop sequentially for Interests under the same FIB entry.
 * If the chosen nexthop is unusable (face down, supression, etc), packet is lost.
 * Initial and retransmitted Interests are treated the same.
 */
#include "api.h"

typedef struct FibEntryInfo
{
  uint8_t nextNexthopIndex;
} FibEntryInfo;

SUBROUTINE uint64_t
RxInterest(SgCtx* ctx)
{
  FibEntryInfo* fei = SgCtx_FibScratchT(ctx, FibEntryInfo);
  if (fei->nextNexthopIndex >= ctx->fibEntry->nNexthops) {
    fei->nextNexthopIndex = 0;
  }

  SgFibNexthopIt it;
  for (SgFibNexthopIt_InitCtx(&it, ctx); SgFibNexthopIt_Valid(&it); SgFibNexthopIt_Next(&it)) {
    if (it.i < fei->nextNexthopIndex) {
      continue;
    }
    fei->nextNexthopIndex = it.i + 1;
    return SgForwardInterest(ctx, it.nh);
  }
  return 9100;
}

uint64_t
SgMain(SgCtx* ctx)
{
  switch (ctx->eventKind) {
    case SGEVT_INTEREST:
      return RxInterest(ctx);
    default:
      return 9000;
  }
}
