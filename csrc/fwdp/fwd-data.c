#include "fwd.h"
#include "strategy.h"

#include "../core/logger.h"
#include "../pcct/pit-iterator.h"

N_LOG_INIT(FwFwd);

__attribute__((nonnull)) static void
FwFwd_DataUnsolicited(FwFwd* fwd, FwFwdCtx* ctx)
{
  N_LOGD("^ drop=unsolicited");
  rte_pktmbuf_free(ctx->pkt);
  ctx->pkt = NULL;
}

__attribute__((nonnull)) static void
FwFwd_DataNeedDigest(FwFwd* fwd, FwFwdCtx* ctx)
{
  int res = rte_ring_enqueue(fwd->crypto, ctx->npkt);
  if (unlikely(res != 0)) {
    N_LOGD("^ error=crypto-enqueue-error-%d", res);
    rte_pktmbuf_free(ctx->pkt);
    NULLize(ctx->pkt);
  } else {
    N_LOGD("^ helper=crypto");
    NULLize(ctx->npkt); // npkt is now owned by FwCrypto
  }
}

__attribute__((nonnull)) static void
FwFwd_DataSatisfy(FwFwd* fwd, FwFwdCtx* ctx)
{
  char debugStringBuffer[PitDebugStringLength];
  N_LOGD("^ pit-entry=%p pit-key=%s", ctx->pitEntry,
         PitEntry_ToDebugString(ctx->pitEntry, debugStringBuffer));

  PitDnIt it;
  for (PitDnIt_Init(&it, ctx->pitEntry); PitDnIt_Valid(&it); PitDnIt_Next(&it)) {
    PitDn* dn = it.dn;
    if (unlikely(dn->face == 0)) {
      if (it.index == 0) {
        N_LOGD("^ drop=PitDn-empty");
      }
      break;
    }
    if (unlikely(dn->expiry < ctx->rxTime)) {
      N_LOGD("^ dn-expired=%" PRI_FaceID, dn->face);
      continue;
    }
    if (unlikely(Face_IsDown(dn->face))) {
      N_LOGD("^ no-data-to=%" PRI_FaceID " drop=face-down", dn->face);
      continue;
    }

    Packet* outNpkt = Packet_Clone(ctx->npkt, &fwd->mp, Face_PacketTxAlign(dn->face));
    N_LOGD("^ data-to=%" PRI_FaceID " npkt=%p dn-token=%016" PRIx64, dn->face, outNpkt, dn->token);
    if (likely(outNpkt != NULL)) {
      struct rte_mbuf* outPkt = Packet_ToMbuf(outNpkt);
      outPkt->port = ctx->rxFace;
      Mbuf_SetTimestamp(outPkt, ctx->rxTime);
      LpL3* lpl3 = Packet_GetLpL3Hdr(outNpkt);
      lpl3->pitToken = dn->token;
      lpl3->congMark = dn->congMark;
      Face_Tx(dn->face, outNpkt);
    }
  }

  if (likely(ctx->fibEntry != NULL)) {
    ++ctx->fibEntryDyn->nRxData;
    uint64_t res = SgInvoke(ctx->fibEntry->strategy, ctx);
    N_LOGD("^ fib-entry-depth=%" PRIu8 " sg-id=%d sg-res=%" PRIu64, ctx->fibEntry->nComps,
           ctx->fibEntry->strategy->id, res);
  }
}

void
FwFwd_RxData(FwFwd* fwd, FwFwdCtx* ctx)
{
  N_LOGD("RxData data-from=%" PRI_FaceID " npkt=%p up-token=%016" PRIx64, ctx->rxFace, ctx->npkt,
         ctx->rxToken);

  PitFindResult pitFound = Pit_FindByData(fwd->pit, ctx->npkt);
  if (PitFindResult_Is(pitFound, PIT_FIND_NONE)) {
    FwFwd_DataUnsolicited(fwd, ctx);
    return;
  }
  if (PitFindResult_Is(pitFound, PIT_FIND_NEED_DIGEST)) {
    FwFwd_DataNeedDigest(fwd, ctx);
    return;
  }

  ctx->nhFlt = ~0; // disallow all forwarding
  rcu_read_lock();

  if (PitFindResult_Is(pitFound, PIT_FIND_PIT0)) {
    ctx->pitEntry = PitFindResult_GetPitEntry0(pitFound);
    FwFwdCtx_SetFibEntry(ctx, PitEntry_FindFibEntry(ctx->pitEntry, fwd->fib));
    FwFwd_DataSatisfy(fwd, ctx);
  }
  if (PitFindResult_Is(pitFound, PIT_FIND_PIT1)) {
    ctx->pitEntry = PitFindResult_GetPitEntry1(pitFound);
    if (likely(ctx->fibEntry == NULL)) {
      FwFwdCtx_SetFibEntry(ctx, PitEntry_FindFibEntry(ctx->pitEntry, fwd->fib));
    }
    // XXX if both PIT entries have the same downstream, Data is sent twice
    FwFwd_DataSatisfy(fwd, ctx);
  }

  NULLize(ctx->fibEntry); // fibEntry is inaccessible upon RCU unlock
  rcu_read_unlock();

  Cs_Insert(fwd->cs, ctx->npkt, pitFound);
  NULLize(ctx->npkt);     // npkt is owned by CS
  NULLize(ctx->pitEntry); // pitEntry is replaced by csEntry
}
