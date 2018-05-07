/** \file
 *  The multicast strategy forwards incoming Interest to all FIB nexthops.
 */
#include "api.h"

inline uint64_t
RxInterest(SgCtx* ctx)
{
  for (uint8_t i = 0; i < ctx->nNexthops; ++i) {
    FaceId nh = ctx->nexthops[i];
    SgForwardInterest(ctx, nh);
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
