/** \file
 *  The round-robin strategy uses each FIB nexthop sequentially for each
 *  retransmitted Interest on the same PIT entry. It skips any unusable
 *  nexthops (e.g. face is down).
 */
#include "api.h"

typedef struct PitEntryInfo
{
  uint8_t nextNexthopIndex;
} PitEntryInfo;

inline uint64_t
RxInterest(SgCtx* ctx)
{
  PitEntryInfo* pei = SgCtx_PitScratchT(ctx, PitEntryInfo);
  if (pei->nextNexthopIndex >= ctx->fibEntry->nNexthops) {
    pei->nextNexthopIndex = 0;
  }

  SgForwardInterestResult res = SGFWDI_BADFACE;
  FaceId nh;
  SgCtx_ForEachNexthop(ctx, i, nh)
  {
    if (i < (int)pei->nextNexthopIndex) {
      continue;
    }
    pei->nextNexthopIndex = i + 1;
    res = SgForwardInterest(ctx, nh);
    if (res == SGFWDI_OK) {
      break;
    }
  }
  return res == SGFWDI_OK ? 0 : 3;
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
