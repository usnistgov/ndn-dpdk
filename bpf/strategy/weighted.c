/**
 * @file
 * The weighted strategy randomly picks a nexthop by assigned weights.
 * If the chosen nexthop is unusable (face down, supression, etc), packet is lost.
 * Initial and retransmitted Interests are treated the same.
 */
#include "api.h"

typedef struct FibEntryInfo
{
  uint8_t weights[FibMaxNexthops];
} FibEntryInfo;

SUBROUTINE uint64_t
RxInterest(SgCtx* ctx)
{
  FibEntryInfo* fei = SgCtx_FibScratchT(ctx, FibEntryInfo);

  uint32_t totalWeight = 0;
  SgFibNexthopIt it;
  for (SgFibNexthopIt_InitCtx(&it, ctx); SgFibNexthopIt_Valid(&it); SgFibNexthopIt_Next(&it)) {
    totalWeight += fei->weights[it.i];
  }
  if (totalWeight == 0) {
    return 9100;
  }

  uint32_t index = SgRandInt(ctx, totalWeight);
  uint32_t accWeight = 0;
  for (SgFibNexthopIt_InitCtx(&it, ctx); SgFibNexthopIt_Valid(&it); SgFibNexthopIt_Next(&it)) {
    accWeight += fei->weights[it.i];
    if (accWeight > index) {
      return SgForwardInterest(ctx, it.nh);
    }
  }
  return 9101;
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

uint64_t
SgInit(SgCtx* ctx)
{
  FibEntryInfo* fei = SgCtx_FibScratchT(ctx, FibEntryInfo);
  SgGetJSONSlice(fei->weights, ctx, "weights", 1);
  return 0;
}

SGINIT_SCHEMA({
  "$schema" : "http://json-schema.org/draft-07/schema#",
  "type" : "object",
  "properties" : {
    "weights" : {
      "description" : "nexthop weights",
      "type" : "array",
      "minItems" : 1,
      "items" : { "type" : "integer", "minimum" : 1, "maximum" : 255 }
    }
  },
  "required" : ["weights"],
  "additionalProperties" : false
});
