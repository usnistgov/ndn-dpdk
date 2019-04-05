#include "fwd.h"
#include "strategy.h"
#include "token.h"

#include "../../container/pcct/pit-dn-up-it.h"
#include "../../core/logger.h"

INIT_ZF_LOG(FwFwd);

static void
FwFwd_TxNacks(FwFwd* fwd,
              PitEntry* pitEntry,
              TscTime now,
              NackReason reason,
              uint8_t nackHopLimit)
{
  PitDnIt it;
  for (PitDnIt_Init(&it, pitEntry); PitDnIt_Valid(&it); PitDnIt_Next(&it)) {
    PitDn* dn = it.dn;
    if (dn->face == FACEID_INVALID) {
      break;
    }
    if (dn->expiry < now) {
      continue;
    }

    if (unlikely(Face_IsDown(dn->face))) {
      ZF_LOGD("^ no-nack-to=%" PRI_FaceId " drop=face-down", dn->face);
      continue;
    }

    Packet* outNpkt = ModifyInterest(pitEntry->npkt,
                                     dn->nonce,
                                     0,
                                     nackHopLimit,
                                     fwd->headerMp,
                                     fwd->guiderMp,
                                     fwd->indirectMp);
    if (unlikely(outNpkt == NULL)) {
      ZF_LOGD("^ no-nack-to=%" PRI_FaceId " drop=alloc-error", dn->face);
      break;
    }

    MakeNack(outNpkt, reason);
    Packet_GetLpL3Hdr(outNpkt)->pitToken = dn->token;
    ZF_LOGD("^ nack-to=%" PRI_FaceId " reason=%s npkt=%p nonce=%08" PRIx32
            " dn-token=%016" PRIx64,
            dn->face,
            NackReason_ToString(reason),
            outNpkt,
            dn->nonce,
            dn->token);
    Face_Tx(dn->face, outNpkt);
  }
}

void
SgReturnNacks(SgCtx* ctx0, SgNackReason reason)
{
  SgContext* ctx = (SgContext*)ctx0;
  assert(ctx->inner.eventKind == SGEVT_INTEREST);

  FwFwd* fwd = ctx->fwd;
  PitEntry* pitEntry = (PitEntry*)ctx->inner.pitEntry;
  TscTime now = rte_get_tsc_cycles();

  FwFwd_TxNacks(fwd, pitEntry, now, (NackReason)reason, 1);
}

typedef struct FwFwdRxNackContext
{
  union
  {
    Packet* npkt;
    struct rte_mbuf* pkt;
  };

  PitEntry* pitEntry;
  PitUp* up;
  int nPending;
  NackReason leastSevereReason;
} FwFwdRxNackContext;

static bool
FwFwd_VerifyNack(FwFwd* fwd, FwFwdRxNackContext* ctx)
{
  if (unlikely(ctx->pitEntry == NULL)) {
    ZF_LOGD("^ drop=no-PIT-entry");
    return false;
  }

  PNack* nack = Packet_GetNackHdr(ctx->npkt);
  ctx->leastSevereReason = nack->lpl3.nackReason;

  PitUpIt it;
  for (PitUpIt_Init(&it, ctx->pitEntry); PitUpIt_Valid(&it);
       PitUpIt_Next(&it)) {
    if (it.up->face == FACEID_INVALID) {
      break;
    }
    if (it.up->face == ctx->pkt->port) {
      ctx->up = it.up;
      continue;
    }
    if (it.up->nack == NackReason_None) {
      ++ctx->nPending;
    } else {
      ctx->leastSevereReason =
        NackReason_GetMin(ctx->leastSevereReason, it.up->nack);
    }
  }
  if (unlikely(ctx->up == NULL)) {
    return false;
  }

  if (unlikely(ctx->up->nonce != nack->interest.nonce)) {
    ZF_LOGD("^ drop=wrong-nonce pit-nonce=%" PRIx32 " up-nonce=%" PRIx32,
            ctx->up->nonce,
            nack->interest.nonce);
    return false;
  }

  return true;
}

static bool
FwFwd_RxNackDuplicate(FwFwd* fwd,
                      FwFwdRxNackContext* ctx,
                      const FibEntry* fibEntry)
{
  TscTime now = rte_get_tsc_cycles();

  uint32_t upNonce = ctx->up->nonce;
  PitUp_AddRejectedNonce(ctx->up, upNonce);
  bool hasAltNonce = PitUp_ChooseNonce(ctx->up, ctx->pitEntry, now, &upNonce);
  if (!hasAltNonce) {
    return false;
  }

  uint32_t upLifetime = PitEntry_GetTxInterestLifetime(ctx->pitEntry, now);
  uint8_t upHopLimit = PitEntry_GetTxInterestHopLimit(ctx->pitEntry);
  Packet* outNpkt = ModifyInterest(ctx->pitEntry->npkt,
                                   upNonce,
                                   upLifetime,
                                   upHopLimit,
                                   fwd->headerMp,
                                   fwd->guiderMp,
                                   fwd->indirectMp);
  if (unlikely(outNpkt == NULL)) {
    ZF_LOGD("^ no-interest-to=%" PRI_FaceId " drop=alloc-error", ctx->up->face);
    return true;
  }

  uint64_t token =
    FwToken_New(fwd->id, Pit_GetEntryToken(fwd->pit, ctx->pitEntry));
  Packet_InitLpL3Hdr(outNpkt)->pitToken = token;
  Packet_ToMbuf(outNpkt)->timestamp = ctx->pkt->timestamp; // for latency stats

  ZF_LOGD("^ interest-to=%" PRI_FaceId " npkt=%p nonce=%08" PRIx32
          " lifetime=%" PRIu32 " hopLimit=%" PRIu8 " up-token=%016" PRIx64,
          ctx->up->face,
          outNpkt,
          upNonce,
          upLifetime,
          upHopLimit,
          token);
  Face_Tx(ctx->up->face, outNpkt);
  if (fibEntry != NULL) {
    ++fibEntry->dyn->nTxInterests;
  }

  PitUp_RecordTx(ctx->up, ctx->pitEntry, now, upNonce, &fwd->suppressCfg);
  return true;
}

void
FwFwd_RxNack(FwFwd* fwd, Packet* npkt)
{
  FwFwdRxNackContext ctx = { 0 };
  ctx.npkt = npkt;
  uint64_t token = Packet_GetLpL3Hdr(npkt)->pitToken;
  PNack* nack = Packet_GetNackHdr(npkt);
  NackReason reason = nack->lpl3.nackReason;
  uint8_t nackHopLimit = nack->interest.hopLimit;

  ZF_LOGD("nack-from=%" PRI_FaceId " npkt=%p up-token=%016" PRIx64
          " reason=%" PRIu8,
          ctx.pkt->port,
          npkt,
          token,
          reason);

  // find PIT entry
  ctx.pitEntry = Pit_FindByNack(fwd->pit, npkt);

  // verify nonce in Nack matches nonce in PitUp
  if (unlikely(!FwFwd_VerifyNack(fwd, &ctx))) {
    if (ctx.pitEntry != NULL) {
      ++fwd->nNackMismatch;
    }
    rte_pktmbuf_free(ctx.pkt);
    return;
  }

  // record NackReason in PitUp
  ctx.up->nack = reason;

  rcu_read_lock();
  const FibEntry* fibEntry = PitEntry_FindFibEntry(ctx.pitEntry, fwd->fib);
  if (likely(fibEntry != NULL)) {
    ++fibEntry->dyn->nRxNacks;
  }

  // Duplicate: record rejected nonce, resend with an alternate nonce if possible
  if (reason == NackReason_Duplicate &&
      FwFwd_RxNackDuplicate(fwd, &ctx, fibEntry)) {
    rte_pktmbuf_free(ctx.pkt);
    rcu_read_unlock();
    return;
  }

  // find FIB entry and invoke strategy
  SgContext sgCtx = { 0 };
  if (likely(fibEntry != NULL)) {
    sgCtx.fwd = fwd;
    sgCtx.rxTime = ctx.pkt->timestamp;
    sgCtx.dnNonce = nack->interest.nonce;
    sgCtx.inner.eventKind = SGEVT_NACK;
    sgCtx.inner.pkt = (const SgPacket*)ctx.pkt;
    sgCtx.inner.fibEntry = (const SgFibEntry*)fibEntry;
    sgCtx.inner.nhFlt = 0; // TODO prevent forwarding to downstream
    sgCtx.inner.pitEntry = (SgPitEntry*)ctx.pitEntry;
    uint64_t res = SgInvoke(fibEntry->strategy, &sgCtx);
    ZF_LOGD("^ fib-entry-depth=%" PRIu8 " sg-id=%d sg-res=%" PRIu64,
            fibEntry->nComps,
            fibEntry->strategy->id,
            res);
  }
  rcu_read_unlock();

  // if there are more pending upstream or strategy retries, wait for them
  if (ctx.nPending > 0 || sgCtx.nForwarded > 0) {
    ZF_LOGD("^ up-pendings=%d sg-forwarded=%d", ctx.nPending, sgCtx.nForwarded);
    return;
  }

  // return Nacks to downstream
  FwFwd_TxNacks(
    fwd, ctx.pitEntry, ctx.pkt->timestamp, ctx.leastSevereReason, nackHopLimit);
  rte_pktmbuf_free(ctx.pkt);

  // erase PIT entry
  Pit_Erase(fwd->pit, ctx.pitEntry);
}
