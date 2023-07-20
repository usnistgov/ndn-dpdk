#include "fwd.h"
#include "strategy.h"
#include "token.h"

#include "../core/logger.h"
#include "../disk/store.h"

N_LOG_INIT(FwFwd);

__attribute__((nonnull)) static FibEntry*
FwFwd_InterestLookupFib(FwFwd* fwd, Packet* npkt, FibNexthopFilter* nhFlt) {
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
FwFwd_InterestRejectNack(FwFwd* fwd, FwFwdCtx* ctx, NackReason reason) {
  ctx->npkt = Nack_FromInterest(ctx->npkt, reason, &fwd->mp, Face_PacketTxAlign(ctx->rxFace));
  if (unlikely(ctx->npkt == NULL)) {
    return;
  }
  Face_Tx(ctx->rxFace, ctx->npkt);
  NULLize(ctx->npkt);
}

__attribute__((nonnull)) static void
FwFwd_InterestForward(FwFwd* fwd, FwFwdCtx* ctx) {
  ctx->dnNonce = Packet_GetInterestHdr(ctx->npkt)->nonce;

  // detect duplicate nonce
  FaceID dupNonce = PitEntry_FindDuplicateNonce(ctx->pitEntry, ctx->dnNonce, ctx->rxFace);
  if (unlikely(dupNonce != 0)) {
    N_LOGD("^ pit-entry=%p drop=duplicate-nonce(%" PRI_FaceID ") nack-to=%" PRI_FaceID,
           ctx->pitEntry, dupNonce, ctx->rxFace);
    FwFwd_InterestRejectNack(fwd, ctx, NackDuplicate);
    ++fwd->nDupNonce;
    return;
  }

  // insert DN record
  PitDn* dn = PitEntry_InsertDn(ctx->pitEntry, fwd->pit, ctx->npkt);
  if (unlikely(dn == NULL)) {
    N_LOGD("^ pit-entry=%p drop=PitDn-full nack-to=%" PRI_FaceID, ctx->pitEntry, ctx->rxFace);
    FwFwd_InterestRejectNack(fwd, ctx, NackCongestion);
    return;
  }
  NULLize(ctx->npkt); // npkt is owned and possibly freed by pitEntry
  N_LOGD("^ pit-entry=%p(%s)", ctx->pitEntry, PitEntry_ToDebugString(ctx->pitEntry));

  uint64_t res = SgInvoke(ctx->fibEntry->strategy, ctx);
  NULLize(ctx->pitEntry); // strategy may have deleted PIT entry via SgReturnNacks
  N_LOGD("^ sg-res=%" PRIu64 " sg-forwarded=%d", res, ctx->nForwarded);
  if (unlikely(ctx->nForwarded == 0)) {
    ++fwd->nSgNoFwd;
  }
}

__attribute__((nonnull)) static void
FwFwd_InterestHitCsMemory(FwFwd* fwd, FwFwdCtx* ctx, CsEntry* csEntry) {
  Packet* outNpkt = Packet_Clone(csEntry->data, &fwd->mp, Face_PacketTxAlign(ctx->rxFace));
  N_LOGD("^ cs-entry-memory=%p data-to=%" PRI_FaceID " npkt=%p dn-token=%s", csEntry, ctx->rxFace,
         outNpkt, LpPitToken_ToString(&ctx->rxToken));
  if (likely(outNpkt != NULL)) {
    struct rte_mbuf* outPkt = Packet_ToMbuf(outNpkt);
    outPkt->port = RTE_MBUF_PORT_INVALID;
    Mbuf_SetTimestamp(outPkt, ctx->rxTime);
    LpL3* lpl3 = Packet_GetLpL3Hdr(outNpkt);
    lpl3->pitToken = ctx->rxToken;
    lpl3->congMark = Packet_GetLpL3Hdr(ctx->npkt)->congMark;
    Face_Tx(ctx->rxFace, outNpkt);
  }
  FwFwdCtx_FreePkt(ctx);
}

__attribute__((nonnull)) static void
FwFwd_InterestHitCsDisk(FwFwd* fwd, FwFwdCtx* ctx, CsEntry* csEntry) {
  struct rte_mbuf* dataBuf = rte_pktmbuf_alloc(fwd->mp.packet);
  if (unlikely(dataBuf == NULL)) {
    N_LOGD("^ cs-entry-disk=%p disk-slot=%" PRIu64 " drop=alloc-err", csEntry, csEntry->diskSlot);
    FwFwdCtx_FreePkt(ctx);
    return;
  }

  N_LOGD("^ cs-entry-disk=%p disk-slot=%" PRIu64 " helper=disk data-npkt=%p", csEntry,
         csEntry->diskSlot, dataBuf);
  DiskStore_GetData(fwd->cs->diskStore, csEntry->diskSlot, ctx->npkt, dataBuf,
                    &csEntry->diskStored);
  NULLize(ctx->npkt);
}

void
FwFwd_RxInterest(FwFwd* fwd, FwFwdCtx* ctx) {
  PInterest* interest = Packet_GetInterestHdr(ctx->npkt);
  NDNDPDK_ASSERT(interest->hopLimit > 0);

  N_LOGD("RxInterest interest-from=%" PRI_FaceID " npkt=%p dn-token=%s", ctx->rxFace, ctx->npkt,
         LpPitToken_ToString(&ctx->rxToken));

  if (unlikely(fwd->cryptoHelper == NULL && interest->name.hasDigestComp)) {
    N_LOGD("^ drop=no-crypto-helper");
    FwFwdCtx_FreePkt(ctx);
    return;
  }

  // query FIB, reply Nack if no FIB match
  rcu_read_lock();
  FwFwdCtx_SetFibEntry(ctx, FwFwd_InterestLookupFib(fwd, ctx->npkt, &ctx->nhFlt));
  if (unlikely(ctx->fibEntry == NULL)) {
    N_LOGD("^ drop=no-FIB-match nack-to=%" PRI_FaceID, ctx->rxFace);
    FwFwd_InterestRejectNack(fwd, ctx, NackNoRoute);
    ++fwd->nNoFibMatch;
    rcu_read_unlock();
    return;
  }
  N_LOGD("^ fh-index=%d fib-entry-depth=%" PRIu8 " sg-id=%d", interest->activeFwHint,
         ctx->fibEntry->nComps, ctx->fibEntry->strategy->id);
  ++ctx->fibEntryDyn->nRxInterests;

  // lookup PIT-CS
  PitInsertResult pitIns = Pit_Insert(fwd->pit, ctx->npkt, ctx->fibEntry);
  switch (pitIns.kind) {
    case PIT_INSERT_PIT: {
      ctx->pitEntry = pitIns.pitEntry;
      FwFwd_InterestForward(fwd, ctx);
      break;
    }
    case PIT_INSERT_CS: {
      switch (pitIns.csEntry->kind) {
        case CsEntryMemory:
          FwFwd_InterestHitCsMemory(fwd, ctx, pitIns.csEntry);
          break;
        case CsEntryDisk:
          FwFwd_InterestHitCsDisk(fwd, ctx, pitIns.csEntry);
          break;
        case CsEntryNone:
        case CsEntryIndirect:
          NDNDPDK_ASSERT(false); // CsEntryNone or CsEntryIndirect isn't a match
          break;
      }
      break;
    }
    case PIT_INSERT_FULL:
      N_LOGD("^ drop=PIT-full nack-to=%" PRI_FaceID, ctx->rxFace);
      FwFwd_InterestRejectNack(fwd, ctx, NackCongestion);
      break;
  }

  NULLize(ctx->fibEntry); // fibEntry is inaccessible upon RCU unlock
  NULLize(ctx->fibEntryDyn);
  rcu_read_unlock();
}

SgForwardInterestResult
SgForwardInterest(SgCtx* ctx0, FaceID nh) {
  FwFwdCtx* ctx = (FwFwdCtx*)ctx0;
  FwFwd* fwd = ctx->fwd;
  TscTime now = rte_get_tsc_cycles();

  if (unlikely(Face_IsDown(nh))) {
    N_LOGD("^ no-interest-to=%" PRI_FaceID " drop=face-down", nh);
    return SGFWDI_BADFACE;
  }

  PitUp* up = PitEntry_ReserveUp(ctx->pitEntry, fwd->pit, nh);
  if (unlikely(up == NULL)) {
    N_LOGD("^ no-interest-to=%" PRI_FaceID " drop=PitUp-full", nh);
    return SGFWDI_ALLOCERR;
  }

  if (PitUp_ShouldSuppress(up, now)) {
    N_LOGD("^ no-interest-to=%" PRI_FaceID " drop=suppressed", nh);
    return SGFWDI_SUPPRESSED;
  }

  InterestGuiders guiders = {
    .nonce = ctx->dnNonce,
    .lifetime = PitEntry_GetTxInterestLifetime(ctx->pitEntry, now),
    .hopLimit = PitEntry_GetTxInterestHopLimit(ctx->pitEntry),
  };
  bool hasNonce = PitUp_ChooseNonce(up, ctx->pitEntry, now, &guiders.nonce);
  if (unlikely(!hasNonce)) {
    N_LOGD("^ no-interest-to=%" PRI_FaceID " drop=nonces-rejected", nh);
    return SGFWDI_NONONCE;
  }
  if (unlikely(guiders.hopLimit == 0)) {
    N_LOGD("^ no-interest-to=%" PRI_FaceID " drop=hoplimit-zero", nh);
    return SGFWDI_HOPZERO;
  }

  Packet* outNpkt =
    Interest_ModifyGuiders(ctx->pitEntry->npkt, guiders, &fwd->mp, Face_PacketTxAlign(nh));
  if (unlikely(outNpkt == NULL)) {
    N_LOGD("^ no-interest-to=%" PRI_FaceID " drop=alloc-err", nh);
    return SGFWDI_ALLOCERR;
  }

  LpPitToken* outToken = &Packet_GetLpL3Hdr(outNpkt)->pitToken;
  FwToken_Set(outToken, fwd->id, PitEntry_GetToken(ctx->pitEntry));
  Mbuf_SetTimestamp(Packet_ToMbuf(outNpkt), ctx->rxTime); // for latency stats

  N_LOGD("^ interest-to=%" PRI_FaceID " npkt=%p " PRI_InterestGuiders " up-token=%s", nh, outNpkt,
         InterestGuiders_Fmt(guiders), LpPitToken_ToString(outToken));
  Face_Tx(nh, outNpkt);
  ++ctx->fibEntryDyn->nTxInterests;

  PitUp_RecordTx(up, ctx->pitEntry, now, guiders.nonce, &fwd->suppressCfg);
  ++ctx->nForwarded;
  return SGFWDI_OK;
}
