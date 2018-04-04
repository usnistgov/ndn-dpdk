#include "api.h"

inline uint64_t
RxInterest(SgCtx* ctx)
{
  for (uint8_t i = 0; i < ctx->nNexthops; ++i) {
    FaceId nh = ctx->nexthops[i];
    ForwardInterest(ctx, nh);
  }
  return 0;
}

uint64_t
Program(SgCtx* ctx)
{
  switch (ctx->eventKind) {
    case SGEVT_INTEREST:
      return RxInterest(ctx);
    default:
      return 2;
  }
}
