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
  FaceId dnFace;

  const FibEntry* fibEntry;
  FibNexthopFilter nhFlt;

  PitEntry* pitEntry;
  CsEntry* csEntry;
} FwFwdRxInterestContext;

static const FibEntry*
FwFwd_InterestLookupFib(FwFwd* fwd, Packet* npkt, FibNexthopFilter* nhFlt)
{
  PInterest* interest = Packet_GetInterestHdr(npkt);
  FaceId dnFace = Packet_ToMbuf(npkt)->port;

  if (likely(interest->nFhs == 0)) {
    const FibEntry* entry = Fib_Lpm(fwd->fib, &interest->name);
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

  for (int fhIndex = 0; fhIndex < interest->nFhs; ++fhIndex) {
    NdnError e = PInterest_SelectActiveFh(interest, fhIndex);
    if (unlikely(e != NdnError_OK)) {
      // caller would treat this as "no FIB match" and reply Nack
      return false;
    }

    const FibEntry* entry = Fib_Lpm(fwd->fib, &interest->activeFhName);
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

static void
FwFwd_InterestForward(FwFwd* fwd, FwFwdRxInterestContext* ctx)
{
  SgContext sgCtx = { 0 };
  sgCtx.rxTime = ctx->pkt->timestamp;
  sgCtx.dnNonce = Packet_GetInterestHdr(ctx->npkt)->nonce;

  // detect duplicate nonce
  FaceId dupNonce =
    PitEntry_FindDuplicateNonce(ctx->pitEntry, sgCtx.dnNonce, ctx->dnFace);
  if (unlikely(dupNonce != FACEID_INVALID)) {
    ZF_LOGD("^ pit-entry=%p drop=duplicate-nonce(%" PRI_FaceId
            ") nack-to=%" PRI_FaceId,
            ctx->pitEntry,
            dupNonce,
            ctx->dnFace);
    MakeNack(ctx->npkt, NackReason_Duplicate);
    Face_Tx(ctx->dnFace, ctx->npkt);
    ++fwd->nDupNonce;
    return;
  }

  // insert DN record
  PitDn* dn = PitEntry_InsertDn(ctx->pitEntry, fwd->pit, ctx->npkt);
  if (unlikely(dn == NULL)) {
    ZF_LOGD("^ pit-entry=%p drop=PitDn-full nack-to=%" PRI_FaceId,
            ctx->pitEntry,
            ctx->dnFace);
    MakeNack(ctx->npkt, NackReason_Congestion);
    Face_Tx(ctx->dnFace, ctx->npkt);
    return;
  }
  ctx->npkt = NULL; // npkt is owned and possibly freed by pitEntry
  ZF_LOGD(
    "^ pit-entry=%p(%s)", ctx->pitEntry, PitEntry_ToDebugString(ctx->pitEntry));

  sgCtx.fwd = fwd;
  sgCtx.inner.eventKind = SGEVT_INTEREST;
  sgCtx.inner.pkt = (const SgPacket*)ctx->pkt;
  sgCtx.inner.fibEntry = (const SgFibEntry*)ctx->fibEntry;
  sgCtx.inner.nhFlt = (SgFibNexthopFilter)ctx->nhFlt;
  sgCtx.inner.pitEntry = (SgPitEntry*)ctx->pitEntry;
  uint64_t res = SgInvoke(ctx->fibEntry->strategy, &sgCtx);
  ctx->pitEntry = NULL; // strategy may have deleted PIT entry via SgReturnNacks
  ZF_LOGD("^ sg-res=%" PRIu64 " sg-forwarded=%d", res, sgCtx.nForwarded);
  if (unlikely(sgCtx.nForwarded == 0)) {
    ++fwd->nSgNoFwd;
  }
}

static void
FwFwd_InterestHitCs(FwFwd* fwd, FwFwdRxInterestContext* ctx)
{
  uint64_t dnToken = Packet_GetLpL3Hdr(ctx->npkt)->pitToken;
  Packet* outNpkt =
    ClonePacket(ctx->csEntry->data, fwd->headerMp, fwd->indirectMp);
  ZF_LOGD("^ cs-entry=%p data-to=%" PRI_FaceId " npkt=%p dn-token=%016" PRIx64,
          ctx->csEntry,
          ctx->dnFace,
          outNpkt,
          dnToken);
  if (likely(outNpkt != NULL)) {
    Packet_GetLpL3Hdr(outNpkt)->pitToken = dnToken;
    Packet_CopyTimestamp(outNpkt, ctx->npkt);
    Face_Tx(ctx->dnFace, outNpkt);
  }
  rte_pktmbuf_free(ctx->pkt);
}

void
FwFwd_RxInterest(FwFwd* fwd, Packet* npkt)
{
  FwFwdRxInterestContext ctx = { 0 };
  ctx.npkt = npkt;
  ctx.dnFace = ctx.pkt->port;
  PInterest* interest = Packet_GetInterestHdr(npkt);
  assert(interest->hopLimit > 0);
  uint64_t dnToken = Packet_GetLpL3Hdr(npkt)->pitToken;

  ZF_LOGD("interest-from=%" PRI_FaceId " npkt=%p dn-token=%016" PRIx64,
          ctx.dnFace,
          npkt,
          dnToken);

  // query FIB, reply Nack if no FIB match
  rcu_read_lock();
  ctx.fibEntry = FwFwd_InterestLookupFib(fwd, ctx.npkt, &ctx.nhFlt);
  if (unlikely(ctx.fibEntry == NULL)) {
    ZF_LOGD("^ drop=no-FIB-match nack-to=%" PRI_FaceId, ctx.dnFace);
    MakeNack(npkt, NackReason_NoRoute);
    Face_Tx(ctx.dnFace, npkt);
    ++fwd->nNoFibMatch;
    rcu_read_unlock();
    return;
  }
  ZF_LOGD("^ fh-index=%d fib-entry-depth=%" PRIu8 " sg-id=%d",
          interest->activeFh,
          ctx.fibEntry->nComps,
          ctx.fibEntry->strategy->id);
  ++ctx.fibEntry->dyn->nRxInterests;

  // lookup PIT-CS
  PitInsertResult pitIns = Pit_Insert(fwd->pit, npkt, ctx.fibEntry);
  switch (PitInsertResult_GetKind(pitIns)) {
    case PIT_INSERT_PIT0:
    case PIT_INSERT_PIT1: {
      ctx.pitEntry = PitInsertResult_GetPitEntry(pitIns);
      FwFwd_InterestForward(fwd, &ctx);
      break;
    }
    case PIT_INSERT_CS: {
      ctx.csEntry = CsEntry_GetDirect(PitInsertResult_GetCsEntry(pitIns));
      FwFwd_InterestHitCs(fwd, &ctx);
      break;
    }
    case PIT_INSERT_FULL:
      ZF_LOGD("^ drop=PIT-full nack-to=%" PRI_FaceId, ctx.dnFace);
      MakeNack(npkt, NackReason_Congestion);
      Face_Tx(ctx.dnFace, npkt);
      break;
    default:
      assert(false); // no other cases
      break;
  }

  rcu_read_unlock();
}

SgForwardInterestResult
SgForwardInterest(SgCtx* ctx0, FaceId nh)
{
  SgContext* ctx = (SgContext*)ctx0;
  FwFwd* fwd = ctx->fwd;
  const FibEntry* fibEntry = (const FibEntry*)ctx->inner.fibEntry;
  PitEntry* pitEntry = (PitEntry*)ctx->inner.pitEntry;
  TscTime now = rte_get_tsc_cycles();

  if (unlikely(Face_IsDown(nh))) {
    ZF_LOGD("^ no-interest-to=%" PRI_FaceId " drop=face-down", nh);
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
  uint8_t upHopLimit = PitEntry_GetTxInterestHopLimit(pitEntry);
  if (unlikely(upHopLimit == 0)) {
    ZF_LOGD("^ no-interest-to=%" PRI_FaceId " drop=hoplimit-zero", nh);
    return SGFWDI_HOPZERO;
  }
  Packet* outNpkt = ModifyInterest(pitEntry->npkt,
                                   upNonce,
                                   upLifetime,
                                   upHopLimit,
                                   fwd->headerMp,
                                   fwd->guiderMp,
                                   fwd->indirectMp);
  if (unlikely(outNpkt == NULL)) {
    ZF_LOGD("^ no-interest-to=%" PRI_FaceId " drop=alloc-error", nh);
    return SGFWDI_ALLOCERR;
  }

  uint64_t token = FwToken_New(fwd->id, Pit_GetEntryToken(fwd->pit, pitEntry));
  Packet_InitLpL3Hdr(outNpkt)->pitToken = token;
  Packet_ToMbuf(outNpkt)->timestamp = ctx->rxTime; // for latency stats

  ZF_LOGD("^ interest-to=%" PRI_FaceId " npkt=%p nonce=%08" PRIx32
          " lifetime=%" PRIu32 " hopLimit=%" PRIu8 " up-token=%016" PRIx64,
          nh,
          outNpkt,
          upNonce,
          upLifetime,
          upHopLimit,
          token);
  Face_Tx(nh, outNpkt);
  ++fibEntry->dyn->nTxInterests;

  PitUp_RecordTx(up, pitEntry, now, upNonce, &fwd->suppressCfg);
  ++ctx->nForwarded;
  return SGFWDI_OK;
}
