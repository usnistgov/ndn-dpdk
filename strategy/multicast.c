/** 
 * @file
 * The multicast strategy forwards incoming Interest to all FIB nexthops.
 */
#include "api.h"

SUBROUTINE uint64_t
RxInterest(SgCtx* ctx)
{
  SgFibNexthopIt it;
  for (SgFibNexthopIt_Init2(&it, ctx); SgFibNexthopIt_Valid(&it);
       SgFibNexthopIt_Next(&it)) {
    SgForwardInterest(ctx, it.nh);
  }
  return 0;
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
