/** \file
 *  The delay strategy delays every incoming Interest by 200 milliseconds,
 *  and then forwards it to the first available nexthop/
 */
#include "api.h"

SUBROUTINE uint64_t
Timer(SgCtx* ctx)
{
  SgFibNexthopIt it;
  for (SgFibNexthopIt_Init2(&it, ctx); SgFibNexthopIt_Valid(&it);
       SgFibNexthopIt_Next(&it)) {
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
  bool ok = SgSetTimer(ctx, SgTscFromMillis(ctx, 200));
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
