#include "fwd.h"
#include "strategy.h"

#include "../../container/pcct/pit-dn-up-it.h"
#include "../../core/logger.h"

INIT_ZF_LOG(FwFwd);

typedef struct FwFwdRxDataContext
{
  union
  {
    Packet* npkt;
    struct rte_mbuf* pkt;
  };
  FaceId upFace;

  const FibEntry* fibEntry;
  PitEntry* pitEntry;
} FwFwdRxDataContext;

static void
FwFwd_DataUnsolicited(FwFwd* fwd, FwFwdRxDataContext* ctx)
{
  ZF_LOGD("^ drop=unsolicited");
  rte_pktmbuf_free(ctx->pkt);
}

static void
FwFwd_DataSatisfy(FwFwd* fwd, FwFwdRxDataContext* ctx)
{
  ZF_LOGD("^ pit-entry=%p pit-key=%s", ctx->pitEntry,
          PitEntry_ToDebugString(ctx->pitEntry));

  PitDnIt it;
  for (PitDnIt_Init(&it, ctx->pitEntry); PitDnIt_Valid(&it);
       PitDnIt_Next(&it)) {
    PitDn* dn = it.dn;
    if (unlikely(dn->face == FACEID_INVALID)) {
      if (index == 0) {
        ZF_LOGD("^ drop=PitDn-empty");
      }
      break;
    }
    if (unlikely(dn->expiry < ctx->pkt->timestamp)) {
      ZF_LOGD("^ dn-expired=%" PRI_FaceId, dn->face);
      continue;
    }
    if (unlikely(Face_IsDown(dn->face))) {
      ZF_LOGD("^ no-data-to=%" PRI_FaceId " drop=face-down", dn->face);
      continue;
    }

    Packet* outNpkt = ClonePacket(ctx->npkt, fwd->headerMp, fwd->indirectMp);
    ZF_LOGD("^ data-to=%" PRI_FaceId " npkt=%p dn-token=%016" PRIx64, dn->face,
            outNpkt, dn->token);
    if (likely(outNpkt != NULL)) {
      Packet_GetLpL3Hdr(outNpkt)->pitToken = dn->token;
      Face_Tx(dn->face, outNpkt);
    }
  }

  if (likely(ctx->fibEntry != NULL)) {
    SgContext sgCtx = { 0 };
    sgCtx.fwd = fwd;
    sgCtx.inner.eventKind = SGEVT_DATA;
    sgCtx.inner.pkt = (const SgPacket*)ctx->pkt;
    sgCtx.inner.fibEntry = (const SgFibEntry*)ctx->fibEntry;
    sgCtx.inner.nhFlt = ~0;
    sgCtx.inner.pitEntry = (SgPitEntry*)ctx->pitEntry;
    uint64_t res = SgInvoke(ctx->fibEntry->strategy, &sgCtx);
    ZF_LOGD("^ fib-entry-depth=%" PRIu8 " sg-id=%d sg-res=%" PRIu64,
            ctx->fibEntry->nComps, ctx->fibEntry->strategy->id, res);
  }
}

void
FwFwd_RxData(FwFwd* fwd, Packet* npkt)
{
  FwFwdRxDataContext ctx = { 0 };
  ctx.npkt = npkt;
  ctx.upFace = ctx.pkt->port;
  uint64_t token = Packet_GetLpL3Hdr(npkt)->pitToken;

  ZF_LOGD("data-from=%" PRI_FaceId " npkt=%p up-token=%016" PRIx64, ctx.upFace,
          npkt, token);

  PitResult pitFound = Pit_FindByData(fwd->pit, npkt);
  switch (PitResult_GetKind(pitFound)) {
    case PIT_FIND_NONE:
      FwFwd_DataUnsolicited(fwd, &ctx);
      return;
    case PIT_FIND_PIT0:
      ctx.pitEntry = PitFindResult_GetPitEntry0(pitFound);
      ctx.fibEntry = PitEntry_FindFibEntry(ctx.pitEntry, fwd->fib);
      FwFwd_DataSatisfy(fwd, &ctx);
      break;
    case PIT_FIND_PIT1:
      ctx.pitEntry = PitFindResult_GetPitEntry1(pitFound);
      ctx.fibEntry = PitEntry_FindFibEntry(ctx.pitEntry, fwd->fib);
      FwFwd_DataSatisfy(fwd, &ctx);
      break;
    case PIT_FIND_PIT01:
      ctx.pitEntry = PitFindResult_GetPitEntry0(pitFound);
      ctx.fibEntry = PitEntry_FindFibEntry(ctx.pitEntry, fwd->fib);
      // XXX if both PIT entries have the same downstream, Data is sent twice
      FwFwd_DataSatisfy(fwd, &ctx);
      ctx.pitEntry = PitFindResult_GetPitEntry1(pitFound);
      FwFwd_DataSatisfy(fwd, &ctx);
      break;
  }

  // insert to CS
  Cs_Insert(fwd->cs, npkt, pitFound);
}
