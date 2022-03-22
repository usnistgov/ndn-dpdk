#include "fwd.h"
#include "strategy.h"
#include "token.h"

#include "../core/logger.h"
#include "../pcct/pit-iterator.h"

N_LOG_INIT(FwFwd);

__attribute__((nonnull)) static void
FwFwd_DataUnsolicited(FwFwd* fwd, FwFwdCtx* ctx)
{
  N_LOGD("^ drop=unsolicited");
  FwFwdCtx_FreePkt(ctx);
}

__attribute__((nonnull)) static void
FwFwd_DataNeedDigest(FwFwd* fwd, FwFwdCtx* ctx)
{
  // if crypto helper is unavailable, Interests with implicit digest should have been dropped
  NDNDPDK_ASSERT(fwd->cryptoHelper != NULL);

  int res = rte_ring_enqueue(fwd->cryptoHelper, ctx->npkt);
  if (unlikely(res != 0)) {
    N_LOGD("^ error=crypto-enqueue-error-%d", res);
    FwFwdCtx_FreePkt(ctx);
  } else {
    N_LOGD("^ helper=crypto");
    NULLize(ctx->npkt); // npkt is now owned by FwCrypto
  }
}

__attribute__((nonnull)) static void
FwFwd_DataSeekFib(FwFwd* fwd, FwFwdCtx* ctx)
{
  FwFwdCtx_SetFibEntry(ctx, PitEntry_FindFibEntry(ctx->pitEntry, fwd->fib));
  if (unlikely(ctx->fibEntryDyn == NULL)) {
    return;
  }

  ++ctx->fibEntryDyn->nRxData;
  PitUp* up = PitEntry_FindUp(ctx->pitEntry, ctx->rxFace);
  if (likely(up != NULL) && likely(up->nTx == 1) &&
      likely(up->nexthopIndex < ctx->fibEntry->nNexthops) &&
      likely(ctx->fibEntry->nexthops[up->nexthopIndex] == ctx->rxFace)) {
    RttValue_Push(&ctx->fibEntryDyn->rtt[up->nexthopIndex], ctx->rxTime - up->lastTx);
  }
}

__attribute__((nonnull)) static void
FwFwd_DataSatisfy(FwFwd* fwd, FwFwdCtx* ctx)
{
  uint8_t upCongMark = Packet_GetLpL3Hdr(ctx->npkt)->congMark;
  N_LOGD("^ pit-entry=%p(%s)", ctx->pitEntry, PitEntry_ToDebugString(ctx->pitEntry));

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
    N_LOGD("^ data-to=%" PRI_FaceID " npkt=%p dn-token=" PRI_LpPitToken, dn->face, outNpkt,
           LpPitToken_Fmt(&dn->token));
    if (unlikely(outNpkt == NULL)) {
      continue;
    }
    struct rte_mbuf* outPkt = Packet_ToMbuf(outNpkt);
    outPkt->port = ctx->rxFace;
    Mbuf_SetTimestamp(outPkt, ctx->rxTime);
    LpL3* lpl3 = Packet_GetLpL3Hdr(outNpkt);
    lpl3->pitToken = dn->token;
    lpl3->congMark = RTE_MAX(dn->congMark, upCongMark);
    Face_Tx(dn->face, outNpkt);
  }

  if (likely(ctx->fibEntry != NULL)) {
    uint64_t res = SgInvoke(ctx->fibEntry->strategy, ctx);
    N_LOGD("^ fib-entry-depth=%" PRIu8 " sg-id=%d sg-res=%" PRIu64, ctx->fibEntry->nComps,
           ctx->fibEntry->strategy->id, res);
  }
}

void
FwFwd_RxData(FwFwd* fwd, FwFwdCtx* ctx)
{
  N_LOGD("RxData data-from=%" PRI_FaceID " npkt=%p up-token=" PRI_LpPitToken, ctx->rxFace,
         ctx->npkt, LpPitToken_Fmt(&ctx->rxToken));
  if (unlikely(ctx->rxToken.length != FwTokenLength)) {
    N_LOGD("^ drop=bad-token-length");
    FwFwdCtx_FreePkt(ctx);
    return;
  }

  PitFindResult pitFound = Pit_FindByData(fwd->pit, ctx->npkt, FwToken_GetPccToken(&ctx->rxToken));
  if (PitFindResult_Is(pitFound, PIT_FIND_NONE)) {
    FwFwd_DataUnsolicited(fwd, ctx);
    return;
  }
  if (PitFindResult_Is(pitFound, PIT_FIND_NEED_DIGEST)) {
    FwFwd_DataNeedDigest(fwd, ctx);
    return;
  }

  ctx->nhFlt = ~0; // disallow any Interest forwarding
  rcu_read_lock();

  if (PitFindResult_Is(pitFound, PIT_FIND_PIT0)) {
    ctx->pitEntry = PitFindResult_GetPitEntry0(pitFound);
    FwFwd_DataSeekFib(fwd, ctx);
    FwFwd_DataSatisfy(fwd, ctx);
  }
  if (PitFindResult_Is(pitFound, PIT_FIND_PIT1)) {
    ctx->pitEntry = PitFindResult_GetPitEntry1(pitFound);
    if (likely(ctx->fibEntry == NULL)) {
      FwFwd_DataSeekFib(fwd, ctx);
    }
    FwFwd_DataSatisfy(fwd, ctx);
  }

  NULLize(ctx->fibEntry); // fibEntry is inaccessible upon RCU unlock
  NULLize(ctx->fibEntryDyn);
  rcu_read_unlock();

  Cs_Insert(fwd->cs, ctx->npkt, pitFound);
  NULLize(ctx->npkt);     // npkt is owned by CS
  NULLize(ctx->pitEntry); // pitEntry is replaced by csEntry
}
