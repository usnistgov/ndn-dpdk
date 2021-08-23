#include "fwd.h"
#include "strategy.h"

#include "../core/logger.h"
#include "../pcct/pit-iterator.h"

N_LOG_INIT(FwFwd);

__attribute__((nonnull)) static void
FwFwd_TxNacks(FwFwd* fwd, PitEntry* pitEntry, TscTime now, NackReason reason, uint8_t nackHopLimit)
{
  PitDnIt it;
  for (PitDnIt_Init(&it, pitEntry); PitDnIt_Valid(&it); PitDnIt_Next(&it)) {
    PitDn* dn = it.dn;
    if (dn->face == 0) {
      break;
    }
    if (dn->expiry < now) {
      continue;
    }

    if (unlikely(Face_IsDown(dn->face))) {
      N_LOGD("^ no-nack-to=%" PRI_FaceID " drop=face-down", dn->face);
      continue;
    }

    InterestGuiders guiders = {
      .nonce = dn->nonce,
      .hopLimit = nackHopLimit,
    };
    PacketTxAlign align = Face_PacketTxAlign(dn->face);
    Packet* output = Interest_ModifyGuiders(pitEntry->npkt, guiders, &fwd->mp, align);
    if (unlikely(output == NULL)) {
      N_LOGD("^ no-nack-to=%" PRI_FaceID " drop=alloc-error", dn->face);
      break;
    }
    output = Nack_FromInterest(output, reason, &fwd->mp, align);
    NDNDPDK_ASSERT(output !=
                   NULL); // cannot fail because Interest_ModifyGuiders result is already aligned

    Packet_GetLpL3Hdr(output)->pitToken = dn->token;
    N_LOGD("^ nack-to=%" PRI_FaceID " reason=%s npkt=%p nonce=%08" PRIx32
           " dn-token=" PRI_LpPitToken,
           dn->face, NackReason_ToString(reason), output, dn->nonce, LpPitToken_Fmt(&dn->token));
    Face_Tx(dn->face, output);
  }
}

void
SgReturnNacks(SgCtx* ctx0, SgNackReason reason)
{
  FwFwdCtx* ctx = (FwFwdCtx*)ctx0;
  NDNDPDK_ASSERT(ctx->eventKind == SGEVT_INTEREST);

  FwFwd_TxNacks(ctx->fwd, ctx->pitEntry, rte_get_tsc_cycles(), (NackReason)reason, 1);
}

__attribute__((nonnull)) static bool
FwFwd_RxNackDuplicate(FwFwd* fwd, FwFwdCtx* ctx)
{
  TscTime now = rte_get_tsc_cycles();
  PitUp* up = ctx->pitUp;
  PitUp_AddRejectedNonce(up, up->nonce);

  InterestGuiders guiders = {
    .nonce = up->nonce,
    .lifetime = PitEntry_GetTxInterestLifetime(ctx->pitEntry, now),
    .hopLimit = PitEntry_GetTxInterestHopLimit(ctx->pitEntry),
  };
  bool hasAltNonce = PitUp_ChooseNonce(up, ctx->pitEntry, now, &guiders.nonce);
  if (!hasAltNonce) {
    return false;
  }

  Packet* outNpkt =
    Interest_ModifyGuiders(ctx->pitEntry->npkt, guiders, &fwd->mp, Face_PacketTxAlign(up->face));
  if (unlikely(outNpkt == NULL)) {
    N_LOGD("^ no-interest-to=%" PRI_FaceID " drop=alloc-error", up->face);
    return true;
  }

  LpPitToken* outToken = &Packet_GetLpL3Hdr(outNpkt)->pitToken;
  FwToken_Set(outToken, fwd->id, PitEntry_GetToken(ctx->pitEntry));
  Mbuf_SetTimestamp(Packet_ToMbuf(outNpkt), Mbuf_GetTimestamp(ctx->pkt)); // for latency stats

  N_LOGD("^ interest-to=%" PRI_FaceID " npkt=%p " PRI_InterestGuiders " up-token=" PRI_LpPitToken,
         up->face, outNpkt, InterestGuiders_Fmt(guiders), LpPitToken_Fmt(outToken));
  Face_Tx(up->face, outNpkt);
  if (ctx->fibEntryDyn != NULL) {
    ++ctx->fibEntryDyn->nTxInterests;
  }

  PitUp_RecordTx(up, ctx->pitEntry, now, guiders.nonce, &fwd->suppressCfg);
  return true;
}

__attribute__((nonnull)) static void
FwFwd_ProcessNack(FwFwd* fwd, FwFwdCtx* ctx)
{
  PNack* nack = Packet_GetNackHdr(ctx->npkt);
  NackReason reason = nack->lpl3.nackReason;
  uint8_t nackHopLimit = nack->interest.hopLimit;

  N_LOGD("RxNack nack-from=%" PRI_FaceID " npkt=%p up-token=" PRI_LpPitToken " reason=%" PRIu8,
         ctx->rxFace, ctx->npkt, LpPitToken_Fmt(&ctx->rxToken), reason);
  if (unlikely(ctx->rxToken.length != FwTokenLength)) {
    N_LOGD("^ drop=bad-token-length");
    return;
  }

  // find PIT entry
  ctx->pitEntry = Pit_FindByNack(fwd->pit, ctx->npkt, FwToken_GetPccToken(&ctx->rxToken));
  if (unlikely(ctx->pitEntry == NULL)) {
    N_LOGD("^ drop=no-PIT-entry");
    return;
  }

  // verify nonce in Nack matches nonce in PitUp
  // count remaining pending upstreams and find least severe Nack reason
  int nPending = 0;
  NackReason leastSevere = reason;
  PitUpIt it;
  for (PitUpIt_Init(&it, ctx->pitEntry); PitUpIt_Valid(&it); PitUpIt_Next(&it)) {
    if (it.up->face == 0) {
      continue;
    }
    if (it.up->face == ctx->rxFace) {
      if (unlikely(it.up->nonce != nack->interest.nonce)) {
        N_LOGD("^ drop=wrong-nonce pit-nonce=%" PRIx32 " up-nonce=%" PRIx32, it.up->nonce,
               nack->interest.nonce);
        break;
      }
      ctx->pitUp = it.up;
      continue;
    }

    if (it.up->nack == NackNone) {
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
  FwFwdCtx_SetFibEntry(ctx, PitEntry_FindFibEntry(ctx->pitEntry, fwd->fib));
  if (likely(ctx->fibEntry != NULL)) {
    ++ctx->fibEntryDyn->nRxNacks;
  }

  // Duplicate: record rejected nonce, resend with an alternate nonce if possible
  if (reason == NackDuplicate && FwFwd_RxNackDuplicate(fwd, ctx)) {
    NULLize(ctx->fibEntry); // fibEntry is inaccessible upon RCU unlock
    rcu_read_unlock();
    return;
  }

  // invoke strategy if FIB entry exists
  if (likely(ctx->fibEntry != NULL)) {
    // TODO set ctx->nhFlt to prevent forwarding to downstream
    uint64_t res = SgInvoke(ctx->fibEntry->strategy, ctx);
    N_LOGD("^ fib-entry-depth=%" PRIu8 " sg-id=%d sg-res=%" PRIu64, ctx->fibEntry->nComps,
           ctx->fibEntry->strategy->id, res);
  }
  NULLize(ctx->fibEntry); // fibEntry is inaccessible upon RCU unlock
  rcu_read_unlock();

  // if there are more pending upstream or strategy retries, wait for them
  if (nPending + ctx->nForwarded > 0) {
    N_LOGD("^ up-pendings=%d sg-forwarded=%d", nPending, ctx->nForwarded);
    return;
  }

  // return Nacks to downstream and erase PIT entry
  FwFwd_TxNacks(fwd, ctx->pitEntry, ctx->rxTime, leastSevere, nackHopLimit);
  Pit_Erase(fwd->pit, ctx->pitEntry);
  NULLize(ctx->pitEntry);
}

void
FwFwd_RxNack(FwFwd* fwd, FwFwdCtx* ctx)
{
  FwFwd_ProcessNack(fwd, ctx);
  rte_pktmbuf_free(ctx->pkt);
  NULLize(ctx->pkt);
}
