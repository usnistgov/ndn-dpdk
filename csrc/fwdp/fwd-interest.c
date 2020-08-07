#include "fwd.h"
#include "strategy.h"
#include "token.h"

#include "../core/logger.h"

INIT_ZF_LOG(FwFwd);

__attribute__((nonnull)) static FibEntry*
FwFwd_InterestLookupFib(FwFwd* fwd, Packet* npkt, FibNexthopFilter* nhFlt)
{
  PInterest* interest = Packet_GetInterestHdr(npkt);
  FaceID dnFace = Packet_ToMbuf(npkt)->port;

  if (likely(interest->nFwHints == 0)) {
    FibEntry* entry = Fib_Lpm(fwd->fib, &interest->name);
    if (unlikely(entry == NULL)) {
      return NULL;
    }
    *nhFlt = 0;
    int nNexthops = FibNexthopFilter_Reject(nhFlt, entry, dnFace);
    if (unlikely(nNexthops == 0)) {
      return NULL;
    }
    return entry;
  }

  for (int i = 0, end = interest->nFwHints; i < end; ++i) {
    if (unlikely(!PInterest_SelectFwHint(interest, i))) {
      // caller would treat this as "no FIB match" and reply Nack
      return false;
    }

    FibEntry* entry = Fib_Lpm(fwd->fib, &interest->fwHint);
    if (unlikely(entry == NULL)) {
      continue;
    }
    *nhFlt = 0;
    int nNexthops = FibNexthopFilter_Reject(nhFlt, entry, dnFace);
    if (unlikely(nNexthops == 0)) {
      continue;
    }
    return entry;
  }
  return NULL;
}

__attribute__((nonnull)) static void
FwFwd_InterestForward(FwFwd* fwd, FwFwdCtx* ctx)
{
  ctx->dnNonce = Packet_GetInterestHdr(ctx->npkt)->nonce;

  // detect duplicate nonce
  FaceID dupNonce = PitEntry_FindDuplicateNonce(ctx->pitEntry, ctx->dnNonce, ctx->rxFace);
  if (unlikely(dupNonce != 0)) {
    ZF_LOGD("^ pit-entry=%p drop=duplicate-nonce(%" PRI_FaceID ") nack-to=%" PRI_FaceID,
            ctx->pitEntry, dupNonce, ctx->rxFace);
    Face_Tx(ctx->rxFace, Nack_FromInterest(ctx->npkt, NackDuplicate));
    ++fwd->nDupNonce;
    return;
  }

  // insert DN record
  PitDn* dn = PitEntry_InsertDn(ctx->pitEntry, fwd->pit, ctx->npkt);
  if (unlikely(dn == NULL)) {
    ZF_LOGD("^ pit-entry=%p drop=PitDn-full nack-to=%" PRI_FaceID, ctx->pitEntry, ctx->rxFace);
    Face_Tx(ctx->rxFace, Nack_FromInterest(ctx->npkt, NackCongestion));
    return;
  }
  FwFwd_NULLize(ctx->npkt); // npkt is owned and possibly freed by pitEntry
  char debugStringBuffer[PitDebugStringLength];
  ZF_LOGD("^ pit-entry=%p(%s)", ctx->pitEntry,
          PitEntry_ToDebugString(ctx->pitEntry, debugStringBuffer));

  uint64_t res = SgInvoke(ctx->fibEntry->strategy, ctx);
  FwFwd_NULLize(ctx->pitEntry); // strategy may have deleted PIT entry via SgReturnNacks
  ZF_LOGD("^ sg-res=%" PRIu64 " sg-forwarded=%d", res, ctx->nForwarded);
  if (unlikely(ctx->nForwarded == 0)) {
    ++fwd->nSgNoFwd;
  }
}

__attribute__((nonnull)) static void
FwFwd_InterestHitCs(FwFwd* fwd, FwFwdCtx* ctx, CsEntry* csEntry)
{
  Packet* outNpkt = Packet_Clone(csEntry->data, fwd->headerMp, fwd->indirectMp);
  ZF_LOGD("^ cs-entry=%p data-to=%" PRI_FaceID " npkt=%p dn-token=%016" PRIx64, csEntry,
          ctx->rxFace, outNpkt, ctx->rxToken);
  if (likely(outNpkt != NULL)) {
    struct rte_mbuf* outPkt = Packet_ToMbuf(outNpkt);
    outPkt->port = MBUF_INVALID_PORT;
    outPkt->timestamp = ctx->rxTime;
    LpL3* lpl3 = Packet_GetLpL3Hdr(outNpkt);
    lpl3->pitToken = ctx->rxToken;
    lpl3->congMark = Packet_GetLpL3Hdr(ctx->npkt)->congMark;
    Face_Tx(ctx->rxFace, outNpkt);
  }
  rte_pktmbuf_free(ctx->pkt);
  FwFwd_NULLize(ctx->pkt);
}

void
FwFwd_RxInterest(FwFwd* fwd, FwFwdCtx* ctx)
{
  PInterest* interest = Packet_GetInterestHdr(ctx->npkt);
  NDNDPDK_ASSERT(interest->hopLimit > 0);

  ZF_LOGD("interest-from=%" PRI_FaceID " npkt=%p dn-token=%016" PRIx64, ctx->rxFace, ctx->npkt,
          ctx->rxToken);

  // query FIB, reply Nack if no FIB match
  rcu_read_lock();
  FwFwdCtx_SetFibEntry(ctx, FwFwd_InterestLookupFib(fwd, ctx->npkt, &ctx->nhFlt));
  if (unlikely(ctx->fibEntry == NULL)) {
    ZF_LOGD("^ drop=no-FIB-match nack-to=%" PRI_FaceID, ctx->rxFace);
    Face_Tx(ctx->rxFace, Nack_FromInterest(ctx->npkt, NackNoRoute));
    ++fwd->nNoFibMatch;
    rcu_read_unlock();
    return;
  }
  ZF_LOGD("^ fh-index=%d fib-entry-depth=%" PRIu8 " sg-id=%d", interest->activeFwHint,
          ctx->fibEntry->nComps, ctx->fibEntry->strategy->id);
  ++ctx->fibEntryDyn->nRxInterests;

  // lookup PIT-CS
  PitInsertResult pitIns = Pit_Insert(fwd->pit, ctx->npkt, ctx->fibEntry);
  switch (PitInsertResult_GetKind(pitIns)) {
    case PIT_INSERT_PIT0:
    case PIT_INSERT_PIT1: {
      ctx->pitEntry = PitInsertResult_GetPitEntry(pitIns);
      FwFwd_InterestForward(fwd, ctx);
      break;
    }
    case PIT_INSERT_CS: {
      CsEntry* csEntry = CsEntry_GetDirect(PitInsertResult_GetCsEntry(pitIns));
      FwFwd_InterestHitCs(fwd, ctx, csEntry);
      break;
    }
    case PIT_INSERT_FULL:
      ZF_LOGD("^ drop=PIT-full nack-to=%" PRI_FaceID, ctx->rxFace);
      Face_Tx(ctx->rxFace, Nack_FromInterest(ctx->npkt, NackCongestion));
      break;
    default:
      NDNDPDK_ASSERT(false); // no other cases
      break;
  }

  FwFwd_NULLize(ctx->fibEntry); // fibEntry is inaccessible upon RCU unlock
  rcu_read_unlock();
}

SgForwardInterestResult
SgForwardInterest(SgCtx* ctx0, FaceID nh)
{
  FwFwdCtx* ctx = (FwFwdCtx*)ctx0;
  FwFwd* fwd = ctx->fwd;
  TscTime now = rte_get_tsc_cycles();

  if (unlikely(Face_IsDown(nh))) {
    ZF_LOGD("^ no-interest-to=%" PRI_FaceID " drop=face-down", nh);
    return SGFWDI_BADFACE;
  }

  PitUp* up = PitEntry_ReserveUp(ctx->pitEntry, fwd->pit, nh);
  if (unlikely(up == NULL)) {
    ZF_LOGD("^ no-interest-to=%" PRI_FaceID " drop=PitUp-full", nh);
    return SGFWDI_ALLOCERR;
  }

  if (PitUp_ShouldSuppress(up, now)) {
    ZF_LOGD("^ no-interest-to=%" PRI_FaceID " drop=suppressed", nh);
    return SGFWDI_SUPPRESSED;
  }

  uint32_t upNonce = ctx->dnNonce;
  bool hasNonce = PitUp_ChooseNonce(up, ctx->pitEntry, now, &upNonce);
  if (unlikely(!hasNonce)) {
    ZF_LOGD("^ no-interest-to=%" PRI_FaceID " drop=nonces-rejected", nh);
    return SGFWDI_NONONCE;
  }

  uint32_t upLifetime = PitEntry_GetTxInterestLifetime(ctx->pitEntry, now);
  uint8_t upHopLimit = PitEntry_GetTxInterestHopLimit(ctx->pitEntry);
  if (unlikely(upHopLimit == 0)) {
    ZF_LOGD("^ no-interest-to=%" PRI_FaceID " drop=hoplimit-zero", nh);
    return SGFWDI_HOPZERO;
  }
  Packet* outNpkt = Interest_ModifyGuiders(ctx->pitEntry->npkt, upNonce, upLifetime, upHopLimit,
                                           fwd->headerMp, fwd->indirectMp);
  if (unlikely(outNpkt == NULL)) {
    ZF_LOGD("^ no-interest-to=%" PRI_FaceID " drop=alloc-error", nh);
    return SGFWDI_ALLOCERR;
  }

  uint64_t token = FwToken_New(fwd->id, PitEntry_GetToken(ctx->pitEntry));
  Packet_GetLpL3Hdr(outNpkt)->pitToken = token;
  Packet_ToMbuf(outNpkt)->timestamp = ctx->rxTime; // for latency stats

  ZF_LOGD("^ interest-to=%" PRI_FaceID " npkt=%p nonce=%08" PRIx32 " lifetime=%" PRIu32
          " hopLimit=%" PRIu8 " up-token=%016" PRIx64,
          nh, outNpkt, upNonce, upLifetime, upHopLimit, token);
  Face_Tx(nh, outNpkt);
  ++ctx->fibEntryDyn->nTxInterests;

  PitUp_RecordTx(up, ctx->pitEntry, now, upNonce, &fwd->suppressCfg);
  ++ctx->nForwarded;
  return SGFWDI_OK;
}
