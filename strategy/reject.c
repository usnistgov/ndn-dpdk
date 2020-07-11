/** 
 * @file
 * The reject strategy responds a Nack against every Interest.
 */
#include "api.h"

SUBROUTINE uint64_t
RxInterest(SgCtx* ctx)
{
  SgReturnNacks(ctx, SgNackNoRoute);
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
