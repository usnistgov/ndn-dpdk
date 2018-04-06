#include "fwd.h"
#include "strategy.h"
#include "token.h"

#include "../../core/logger.h"

INIT_ZF_LOG(FwFwd);

typedef struct FwFwdRxInterestContext
{
  union
  {
    Packet* npkt;
    struct rte_mbuf* pkt;
  };
  Face* dnFace;

  PitEntry* pitEntry;
  CsEntry* csEntry;

  FaceId nexthops[FIB_ENTRY_MAX_NEXTHOPS];
  uint8_t nNexthops;
} FwFwdRxInterestContext;

static const FibEntry*
FwFwd_LookupFib(FwFwd* fwd, FwFwdRxInterestContext* ctx)
{
  PInterest* interest = Packet_GetInterestHdr(ctx->npkt);

  if (likely(interest->nFhs == 0)) {
    const FibEntry* fibEntry = Fib_Lpm(fwd->fib, &interest->name);
    if (unlikely(fibEntry == NULL)) {
      return NULL;
    }
    ctx->nNexthops =
      FibEntry_FilterNexthops(fibEntry, ctx->nexthops, &ctx->dnFace->id, 1);
    if (unlikely(ctx->nNexthops == 0)) {
      return NULL;
    }
    return fibEntry;
  }

  for (int fhIndex = 0; fhIndex < interest->nFhs; ++fhIndex) {
    NdnError e = PInterest_SelectActiveFh(interest, fhIndex);
    if (unlikely(e != NdnError_OK)) {
      ZF_LOGD("^ drop=bad-fh(%d,%d)", fhIndex, e);
      // caller would treat this as "no FIB match" and reply Nack
      return NULL;
    }

    const FibEntry* fibEntry = Fib_Lpm(fwd->fib, &interest->name);
    if (unlikely(fibEntry == NULL)) {
      continue;
    }
    ctx->nNexthops =
      FibEntry_FilterNexthops(fibEntry, ctx->nexthops, &ctx->dnFace->id, 1);
    if (unlikely(ctx->nNexthops == 0)) {
      continue;
    }
    return fibEntry;
  }

  return NULL;
}

static void
FwFwd_InterestMissCs(FwFwd* fwd, FwFwdRxInterestContext* ctx)
{
  // TODO detect duplicate nonce
  SgContext sgCtx = { 0 };
  sgCtx.rxTime = ctx->pkt->timestamp;
  sgCtx.dnNonce = Packet_GetInterestHdr(ctx->npkt)->nonce;

  // insert DN record
  PitDn* dn = PitEntry_InsertDn(ctx->pitEntry, fwd->pit, ctx->npkt);
  if (unlikely(dn == NULL)) {
    // TODO allocate another entry for excess DN records
    ZF_LOGD("^ pit-entry=%p drop=PitDn-full", ctx->pitEntry);
    rte_pktmbuf_free(ctx->pkt);
    return;
  }
  ctx->npkt = NULL; // npkt is owned and possibly freed by pitEntry
  ZF_LOGD("^ pit-entry=%p pit-key=%s", ctx->pitEntry,
          PitEntry_ToDebugString(ctx->pitEntry));

  sgCtx.fwd = fwd;
  sgCtx.inner.eventKind = SGEVT_INTEREST;
  sgCtx.inner.pitEntry = (SgPitEntry*)ctx->pitEntry;
  sgCtx.inner.nexthops = ctx->nexthops;
  sgCtx.inner.nNexthops = ctx->nNexthops;
  uint64_t res = (*fwd->strategy)(&sgCtx, sizeof(SgCtx));
  ZF_LOGD("^ sg-res=%" PRIu64, res);
}

static void
FwFwd_InterestHitCs(FwFwd* fwd, FwFwdRxInterestContext* ctx)
{
  uint64_t dnToken = Packet_GetLpL3Hdr(ctx->npkt)->pitToken;
  Packet* outNpkt =
    ClonePacket(ctx->csEntry->data, fwd->headerMp, fwd->indirectMp);
  ZF_LOGD("^ cs-entry=%p data-to=%" PRI_FaceId " npkt=%p dn-token=%016" PRIx64,
          ctx->csEntry, ctx->dnFace->id, outNpkt, dnToken);
  if (likely(outNpkt != NULL)) {
    Packet_GetLpL3Hdr(outNpkt)->pitToken = dnToken;
    Packet_CopyTimestamp(outNpkt, ctx->npkt);
    Face_Tx(ctx->dnFace, outNpkt);
  }
}

void
FwFwd_RxInterest(FwFwd* fwd, Packet* npkt)
{
  FwFwdRxInterestContext ctx = { 0 };
  ctx.npkt = npkt;
  ctx.dnFace = FaceTable_GetFace(fwd->ft, ctx.pkt->port);
  assert(ctx.dnFace != NULL); // XXX could fail if face fails during forwarding
  PInterest* interest = Packet_GetInterestHdr(npkt);
  uint64_t dnToken = Packet_GetLpL3Hdr(npkt)->pitToken;

  ZF_LOGD("interest-from=%" PRI_FaceId " npkt=%p dn-token=%016" PRIx64,
          ctx.dnFace->id, npkt, dnToken);

  // query FIB, reply Nack if no FIB match
  rcu_read_lock();
  const FibEntry* fibEntry = FwFwd_LookupFib(fwd, &ctx);
  if (unlikely(fibEntry == NULL)) {
    ZF_LOGD("^ drop=no-FIB-match nack-to=%" PRI_FaceId, ctx.dnFace->id);
    MakeNack(npkt, NackReason_NoRoute);
    Face_Tx(ctx.dnFace, npkt);
    rcu_read_unlock();
    return;
  }
  ZF_LOGD("^ fib-entry-depth=%" PRIu8 " nexthop-count=%" PRIu8,
          fibEntry->nComps, ctx.nNexthops);
  assert(ctx.nNexthops > 0);
  fibEntry = NULL;
  rcu_read_unlock();

  // lookup PIT-CS
  PitResult pitIns = Pit_Insert(fwd->pit, npkt);
  switch (PitResult_GetKind(pitIns)) {
    case PIT_INSERT_PIT0:
    case PIT_INSERT_PIT1: {
      ctx.pitEntry = PitInsertResult_GetPitEntry(pitIns);
      FwFwd_InterestMissCs(fwd, &ctx);
      break;
    }
    case PIT_INSERT_CS: {
      ctx.csEntry = PitInsertResult_GetCsEntry(pitIns);
      FwFwd_InterestHitCs(fwd, &ctx);
      break;
    }
    case PIT_INSERT_FULL:
      ZF_LOGD("^ drop=PIT-full nack-to=%" PRI_FaceId, ctx.dnFace->id);
      MakeNack(npkt, NackReason_Congestion);
      Face_Tx(ctx.dnFace, npkt);
      break;
    default:
      assert(false); // no other cases
      break;
  }
}

SgForwardInterestResult
SgForwardInterest(SgCtx* ctx0, FaceId nh)
{
  SgContext* ctx = (SgContext*)ctx0;
  FwFwd* fwd = ctx->fwd;
  PitEntry* pitEntry = (PitEntry*)ctx->inner.pitEntry;
  TscTime now = rte_get_tsc_cycles();

  Face* outFace = FaceTable_GetFace(fwd->ft, nh);
  if (unlikely(outFace == NULL)) {
    ZF_LOGD("^ no-interest-to=%" PRI_FaceId " drop=no-face", nh);
    return SGFWDI_BADFACE;
  }

  PitUp* up = PitEntry_ReserveUp(pitEntry, fwd->pit, nh);
  if (unlikely(up == NULL)) {
    ZF_LOGD("^ no-interest-to=%" PRI_FaceId " drop=PitUp-full", nh);
    return SGFWDI_ALLOCERR;
  }

  if (PitUp_ShouldSuppress(up, now)) {
    ZF_LOGD("^ no-interest-to=%" PRI_FaceId " drop=suppressed", nh);
    return SGFWDI_SUPPRESSED;
  }

  uint32_t upNonce = ctx->dnNonce;
  bool hasNonce = PitUp_ChooseNonce(up, pitEntry, now, &upNonce);
  if (unlikely(!hasNonce)) {
    ZF_LOGD("^ no-interest-to=%" PRI_FaceId " drop=nonces-rejected", nh);
    return SGFWDI_NONONCE;
  }

  uint32_t upLifetime = PitEntry_GetTxInterestLifetime(pitEntry, now);
  uint8_t hopLimit = 0xFF; // TODO properly set HopLimit
  Packet* outNpkt =
    ModifyInterest(pitEntry->npkt, upNonce, upLifetime, hopLimit, fwd->headerMp,
                   fwd->guiderMp, fwd->indirectMp);
  if (unlikely(outNpkt == NULL)) {
    ZF_LOGD("^ no-interest-to=%" PRI_FaceId " drop=alloc-error", nh);
    return SGFWDI_ALLOCERR;
  }

  uint64_t token = FwToken_New(fwd->id, Pit_GetEntryToken(fwd->pit, pitEntry));
  Packet_InitLpL3Hdr(outNpkt)->pitToken = token;
  Packet_ToMbuf(outNpkt)->timestamp = ctx->rxTime; // for latency stats

  ZF_LOGD("^ interest-to=%" PRI_FaceId " npkt=%p nonce=%08" PRIx32
          " up-token=%016" PRIx64,
          nh, outNpkt, upNonce, token);
  Face_Tx(outFace, outNpkt);

  PitUp_RecordTx(up, pitEntry, now, upNonce, &fwd->suppressCfg);
  return SGFWDI_OK;
}
