/**
 * @file
 * The delay strategy delays every Interest then forwards it to the first available nexthop.
 * Delay duration is configured via FibEntryInfo.
 */
#include "api.h"

typedef struct FibEntryInfo
{
  uint32_t delay; ///< delay duration in milliseconds
} FibEntryInfo;

SUBROUTINE uint64_t
Timer(SgCtx* ctx)
{
  SgFibNexthopIt it;
  for (SgFibNexthopIt_InitCtx(&it, ctx); SgFibNexthopIt_Valid(&it); SgFibNexthopIt_Next(&it)) {
    SgForwardInterestResult res = SgForwardInterest(ctx, it.nh);
    if (res == SGFWDI_OK) {
      return 0;
    }
  }
  return 3;
}

SUBROUTINE uint64_t
RxInterest(SgCtx* ctx)
{
  FibEntryInfo* fei = SgCtx_FibScratchT(ctx, FibEntryInfo);
  bool ok = SgSetTimer(ctx, SgTscFromMillis(ctx, fei->delay));
  return ok ? 0 : 3;
}

uint64_t
SgMain(SgCtx* ctx)
{
  switch (ctx->eventKind) {
    case SGEVT_TIMER:
      return Timer(ctx);
    case SGEVT_INTEREST:
      return RxInterest(ctx);
    default:
      return 2;
  }
}
