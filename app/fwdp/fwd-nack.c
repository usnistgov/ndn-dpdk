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
  FwFwdCtx* ctx = (FwFwdCtx*)ctx0;
  assert(ctx->eventKind == SGEVT_INTEREST);

  FwFwd_TxNacks(
    ctx->fwd, ctx->pitEntry, rte_get_tsc_cycles(), (NackReason)reason, 1);
}

static bool
FwFwd_RxNackDuplicate(FwFwd* fwd, FwFwdCtx* ctx)
{
  TscTime now = rte_get_tsc_cycles();

  uint32_t upNonce = ctx->pitUp->nonce;
  PitUp_AddRejectedNonce(ctx->pitUp, upNonce);
  bool hasAltNonce =
    PitUp_ChooseNonce(ctx->pitUp, ctx->pitEntry, now, &upNonce);
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
    ZF_LOGD("^ no-interest-to=%" PRI_FaceId " drop=alloc-error",
            ctx->pitUp->face);
    return true;
  }

  uint64_t token =
    FwToken_New(fwd->id, Pit_GetEntryToken(fwd->pit, ctx->pitEntry));
  Packet_InitLpL3Hdr(outNpkt)->pitToken = token;
  Packet_ToMbuf(outNpkt)->timestamp = ctx->pkt->timestamp; // for latency stats

  ZF_LOGD("^ interest-to=%" PRI_FaceId " npkt=%p nonce=%08" PRIx32
          " lifetime=%" PRIu32 " hopLimit=%" PRIu8 " up-token=%016" PRIx64,
          ctx->pitUp->face,
          outNpkt,
          upNonce,
          upLifetime,
          upHopLimit,
          token);
  Face_Tx(ctx->pitUp->face, outNpkt);
  if (ctx->fibEntry != NULL) {
    ++ctx->fibEntry->nTxInterests;
  }

  PitUp_RecordTx(ctx->pitUp, ctx->pitEntry, now, upNonce, &fwd->suppressCfg);
  return true;
}

static void
FwFwd_ProcessNack(FwFwd* fwd, FwFwdCtx* ctx)
{
  PNack* nack = Packet_GetNackHdr(ctx->npkt);
  NackReason reason = nack->lpl3.nackReason;
  uint8_t nackHopLimit = nack->interest.hopLimit;

  ZF_LOGD("nack-from=%" PRI_FaceId " npkt=%p up-token=%016" PRIx64
          " reason=%" PRIu8,
          ctx->rxFace,
          ctx->npkt,
          ctx->rxToken,
          reason);

  // find PIT entry
  ctx->pitEntry = Pit_FindByNack(fwd->pit, ctx->npkt);
  if (unlikely(ctx->pitEntry == NULL)) {
    ZF_LOGD("^ drop=no-PIT-entry");
    return;
  }

  // verify nonce in Nack matches nonce in PitUp
  // count remaining pending upstreams and find least severe Nack reason
  int nPending = 0;
  NackReason leastSevere = reason;
  PitUpIt it;
  for (PitUpIt_Init(&it, ctx->pitEntry); PitUpIt_Valid(&it);
       PitUpIt_Next(&it)) {
    if (it.up->face == FACEID_INVALID) {
      continue;
    }
    if (it.up->face == ctx->rxFace) {
      if (unlikely(it.up->nonce != nack->interest.nonce)) {
        ZF_LOGD("^ drop=wrong-nonce pit-nonce=%" PRIx32 " up-nonce=%" PRIx32,
                it.up->nonce,
                nack->interest.nonce);
        break;
      }
      ctx->pitUp = it.up;
      continue;
    }

    if (it.up->nack == NackReason_None) {
      ++nPending;
    } else {
      leastSevere = NackReason_GetMin(leastSevere, it.up->nack);
    }
  }
  if (unlikely(ctx->pitUp == NULL)) {
    ++fwd->nNackMismatch;
    return;
  }

  // record NackReason in PitUp
  ctx->pitUp->nack = reason;

  // find FIB entry; FIB entry is optional for Nack processing
  rcu_read_lock();
  ctx->fibEntry = PitEntry_FindFibEntry(ctx->pitEntry, fwd->fib);
  if (likely(ctx->fibEntry != NULL)) {
    ++ctx->fibEntry->nRxNacks;
  }

  // Duplicate: record rejected nonce, resend with an alternate nonce if possible
  if (reason == NackReason_Duplicate && FwFwd_RxNackDuplicate(fwd, ctx)) {
    FwFwd_NULLize(ctx->fibEntry); // fibEntry is inaccessible upon RCU unlock
    rcu_read_unlock();
    return;
  }

  // invoke strategy if FIB entry exists
  if (likely(ctx->fibEntry != NULL)) {
    // TODO set ctx->nhFlt to prevent forwarding to downstream
    uint64_t res = SgInvoke(ctx->fibEntry->strategy, ctx);
    ZF_LOGD("^ fib-entry-depth=%" PRIu8 " sg-id=%d sg-res=%" PRIu64,
            ctx->fibEntry->nComps,
            ctx->fibEntry->strategy->id,
            res);
  }
  FwFwd_NULLize(ctx->fibEntry); // fibEntry is inaccessible upon RCU unlock
  rcu_read_unlock();

  // if there are more pending upstream or strategy retries, wait for them
  if (nPending + ctx->nForwarded > 0) {
    ZF_LOGD("^ up-pendings=%d sg-forwarded=%d", nPending, ctx->nForwarded);
    return;
  }

  // return Nacks to downstream and erase PIT entry
  FwFwd_TxNacks(fwd, ctx->pitEntry, ctx->rxTime, leastSevere, nackHopLimit);
  Pit_Erase(fwd->pit, ctx->pitEntry);
  FwFwd_NULLize(ctx->pitEntry);
}

void
FwFwd_RxNack(FwFwd* fwd, FwFwdCtx* ctx)
{
  FwFwd_ProcessNack(fwd, ctx);
  rte_pktmbuf_free(ctx->pkt);
  FwFwd_NULLize(ctx->pkt);
}
