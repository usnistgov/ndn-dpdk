#include "producer.h"

#include "../core/logger.h"

N_LOG_INIT(Tgp);

typedef struct TgpBurstCtx {
  Tgp* p;
  PacketMempools mp;
  PacketTxAlign faceTxAlign;
  TscTime now;
  PktQueuePopResult pop;
  uint16_t nDiscard;
  uint16_t nTx;
  Packet* rx[MaxBurstSize];
  Packet* tx[MaxBurstSize];
} TgpBurstCtx;

__attribute__((nonnull(1))) static inline void
TgpBurstCtx_Tx(TgpBurstCtx* ctx, Packet* npkt) {
  if (unlikely(npkt == NULL)) {
    ++ctx->p->nAllocError;
    return;
  }
  Mbuf_SetTimestamp(Packet_ToMbuf(npkt), ctx->now);
  ctx->tx[ctx->nTx++] = npkt;
}

__attribute__((nonnull)) static inline void
TgpBurstCtx_Discard(TgpBurstCtx* ctx, uint16_t i) {
  NDNDPDK_ASSERT(i >= ctx->nDiscard);
  ctx->rx[ctx->nDiscard++] = ctx->rx[i];
}

__attribute__((nonnull)) static void
Tgp_RespondData(TgpBurstCtx* ctx, uint16_t i, TgpReply* reply) {
  Packet* npkt = ctx->rx[i];
  const PName* interestName = &Packet_GetInterestHdr(npkt)->name;
  LName dataPrefix = PName_ToLName(interestName);
  if (unlikely(interestName->hasDigestComp)) {
    dataPrefix.length -= ImplicitDigestSize;
  }

  Packet* output = DataGen_Encode(&reply->dataGen, dataPrefix, &ctx->mp, ctx->faceTxAlign);
  if (likely(output != NULL)) {
    Packet_GetLpL3Hdr(output)->pitToken = Packet_GetLpL3Hdr(npkt)->pitToken;
  }
  TgpBurstCtx_Tx(ctx, output);
  TgpBurstCtx_Discard(ctx, i);
}

__attribute__((nonnull)) static void
Tgp_RespondNack(TgpBurstCtx* ctx, uint16_t i, TgpReply* reply) {
  Packet* npkt = ctx->rx[i];
  npkt = Nack_FromInterest(npkt, reply->nackReason, &ctx->mp, ctx->faceTxAlign);
  TgpBurstCtx_Tx(ctx, npkt);
}

__attribute__((nonnull)) static void
Tgp_RespondTimeout(TgpBurstCtx* ctx, uint16_t i, __rte_unused TgpReply* reply) {
  TgpBurstCtx_Discard(ctx, i);
}

typedef void (*Tgp_Respond)(TgpBurstCtx* ctx, uint16_t i, TgpReply* reply);

static const Tgp_Respond Tgp_RespondJmp[] = {
  [TgpReplyData] = Tgp_RespondData,
  [TgpReplyNack] = Tgp_RespondNack,
  [TgpReplyTimeout] = Tgp_RespondTimeout,
};

__attribute__((nonnull)) static inline void
Tgp_ProcessInterest(Tgp* p, TgpBurstCtx* ctx, uint16_t i) {
  Packet* npkt = ctx->rx[i];
  int patternID = LNamePrefixFilter_Find(PName_ToLName(&Packet_GetInterestHdr(npkt)->name),
                                         TgpMaxPatterns, p->prefixL, p->prefixV);
  if (unlikely(patternID < 0)) {
    const LpPitToken* token = &Packet_GetLpL3Hdr(npkt)->pitToken;
    N_LOGD(">I dn-token=%s no-pattern", LpPitToken_ToString(token));
    ++p->nNoMatch;
    TgpBurstCtx_Discard(ctx, i);
    return;
  }
  TgpPattern* pattern = &p->pattern[patternID];
  uint8_t replyID = pattern->weight[pcg32_boundedrand_r(&p->replyRng, pattern->nWeights)];
  TgpReply* reply = &pattern->reply[replyID];

  const LpPitToken* token = &Packet_GetLpL3Hdr(npkt)->pitToken;
  N_LOGD(">I dn-token=%s pattern=%d reply=%" PRIu8, LpPitToken_ToString(token), patternID, replyID);
  ++reply->nInterests;
  Tgp_RespondJmp[reply->kind](ctx, i, reply);
}

int
Tgp_Run(Tgp* p) {
  TgpBurstCtx ctx = {
    .p = p,
    .mp = p->mp,
    .faceTxAlign = Face_PacketTxAlign(p->face),
  };
  while (ThreadCtrl_Continue(p->ctrl, ctx.pop.count)) {
    ctx.now = rte_get_tsc_cycles();
    ctx.pop = PktQueue_Pop(&p->rxQueue, (struct rte_mbuf**)ctx.rx, MaxBurstSize, ctx.now);
    if (unlikely(ctx.pop.count == 0)) {
      continue;
    }

    ctx.nDiscard = 0;
    ctx.nTx = 0;
    for (uint16_t i = 0; i < ctx.pop.count; ++i) {
      NDNDPDK_ASSERT(Packet_GetType(ctx.rx[i]) == PktInterest);
      Tgp_ProcessInterest(p, &ctx, i);
    }

    N_LOGD("burst face=%" PRI_FaceID "nRx=%" PRIu16 " nTx=%" PRIu16, p->face, ctx.pop.count,
           ctx.nTx);
    Face_TxBurst(p->face, ctx.tx, ctx.nTx);
    if (likely(ctx.nDiscard > 0)) {
      rte_pktmbuf_free_bulk((struct rte_mbuf**)ctx.rx, ctx.nDiscard);
    }
  }
  return 0;
}
