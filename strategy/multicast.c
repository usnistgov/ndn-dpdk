/** \file
 *  The multicast strategy forwards incoming Interest to all FIB nexthops.
 */
#include "api.h"

inline uint64_t
RxInterest(SgCtx* ctx)
{
  FaceId nh;
  SgCtx_ForEachNexthop(ctx, i, nh) { SgForwardInterest(ctx, nh); }
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
