#include "tx.h"
#include "../core/logger.h"

STATIC_ASSERT_FUNC_TYPE(TgcTxPattern_MakeSuffix, TgcTxPattern_MakeSuffix_Digest);
STATIC_ASSERT_FUNC_TYPE(TgcTxPattern_MakeSuffix, TgcTxPattern_MakeSuffix_Offset);
STATIC_ASSERT_FUNC_TYPE(TgcTxPattern_MakeSuffix, TgcTxPattern_MakeSuffix_Increment);

N_LOG_INIT(Tgc);

__attribute__((nonnull, returns_nonnull)) static __rte_always_inline unaligned_uint64_t*
TgcTxDigestPattern_DataSeqNum(TgcTxPattern* pattern) {
  TgcTxDigestPattern* dp = pattern->digest;
  return RTE_PTR_ADD(dp->prefix.value, pattern->tpl.prefixL + TgcSeqNumSize - sizeof(uint64_t));
}

void
TgcTxDigestPattern_Fill(TgcTxPattern* pattern) {
  TgcTxDigestPattern* dp = pattern->digest;
  Packet* npkts[TgcDigestBurstSize];
  struct rte_crypto_op* ops[TgcDigestBurstSize];

  for (int i = 0; i < TgcDigestBurstSize; ++i) {
    ++*TgcTxDigestPattern_DataSeqNum(pattern);
    Packet* npkt =
      DataGen_Encode(&dp->dataGen, dp->prefix, &dp->dataMp, (PacketTxAlign){.linearize = false});
    *Packet_GetLpL3Hdr(npkt) = (const LpL3){0};
    bool ok = Packet_ParseL3(npkt, ParseForApp);
    NDNDPDK_ASSERT(ok && Packet_GetType(npkt) == PktData);
    npkts[i] = npkt;
    ops[i] = DataDigest_Prepare(&dp->cqp, npkt);
  }

  uint16_t nRej = DataDigest_Enqueue(&dp->cqp, ops, TgcDigestBurstSize);
  if (unlikely(nRej > 0)) {
    N_LOGW("digest-fill error=enqueue-reject count=%" PRIu16 " nRej=%" PRIu16, TgcDigestBurstSize,
           nRej);
    rte_pktmbuf_free_bulk((struct rte_mbuf**)&npkts[TgcDigestBurstSize - nRej], nRej);
    return;
  }
  N_LOGD("digest-fill count=%" PRIu16, TgcDigestBurstSize);
}

uint16_t
TgcTxPattern_MakeSuffix_Digest(TgcTx* ct, uint8_t patternID, TgcTxPattern* pattern) {
  TgcTxDigestPattern* dp = pattern->digest;
  if (unlikely(*TgcTxDigestPattern_DataSeqNum(pattern) - pattern->seqNumV <=
               TgcDigestLowWatermark)) {
    TgcTxDigestPattern_Fill(pattern);
  }

  struct rte_crypto_op* op = NULL;
  uint16_t nDeq = rte_cryptodev_dequeue_burst(dp->cqp.dev, dp->cqp.qp, &op, 1);
  if (unlikely(nDeq == 0)) {
    N_LOGW("digest-pull error=dequeue-empty");
    return 0;
  }
  Packet* npkt = NULL;
  if (unlikely(!DataDigest_Finish(op, &npkt))) {
    N_LOGW("digest-pull error=digest-fail");
    rte_pktmbuf_free(Packet_ToMbuf(npkt));
    return 0;
  }

  PData* data = Packet_GetDataHdr(npkt);
  unaligned_uint64_t* dataSeqNum = RTE_PTR_ADD(
    PName_ToLName(&data->name).value, pattern->tpl.prefixL + TgcSeqNumSize - sizeof(uint64_t));
  pattern->seqNumV = *dataSeqNum;

  static_assert(ImplicitDigestLength == 32, "");
  rte_mov32(pattern->digestV, data->digest);
  rte_pktmbuf_free(Packet_ToMbuf(npkt));
  return TgcSeqNumSize + ImplicitDigestSize;
}

uint16_t
TgcTxPattern_MakeSuffix_Offset(TgcTx* ct, uint8_t patternID, TgcTxPattern* pattern) {
  uint64_t seqNum = ct->pattern[patternID - 1].seqNumV - pattern->seqNumOffset;
  if (unlikely(pattern->seqNumV - seqNum <= UINT32_MAX)) { // same seqNum already requested
    seqNum = pattern->seqNumV + 1;
  }
  pattern->seqNumV = seqNum;
  return TgcSeqNumSize;
}

uint16_t
TgcTxPattern_MakeSuffix_Increment(TgcTx* ct, uint8_t patternID, TgcTxPattern* pattern) {
  ++pattern->seqNumV;
  return TgcSeqNumSize;
}

__attribute__((nonnull)) static __rte_always_inline uint8_t
TgcTx_SelectPattern(TgcTx* ct) {
  uint32_t w = pcg32_boundedrand_r(&ct->trafficRng, ct->nWeights);
  return ct->weight[w];
}

__attribute__((nonnull)) static __rte_always_inline bool
TgcTx_MakeInterest(TgcTx* ct, struct rte_mbuf* pkt, TscTime now) {
  uint8_t id = TgcTx_SelectPattern(ct);
  TgcTxPattern* pattern = &ct->pattern[id];
  ++pattern->nInterests;

  uint16_t suffixL = pattern->makeSuffix(ct, id, pattern);
  if (unlikely(suffixL == 0)) {
    N_LOGW("error pattern=%" PRIu8, id);
    return false;
  }

  LName suffix = (LName){.length = suffixL, .value = &pattern->seqNumT};
  uint32_t nonce = pcg32_random_r(&ct->nonceRng);
  Packet* npkt = InterestTemplate_Encode(&pattern->tpl, pkt, suffix, nonce);
  TgcToken_Set(&Packet_GetLpL3Hdr(npkt)->pitToken, id, ct->runNum, now);
  N_LOGD("<I pattern=%" PRIu8 " seq=%" PRIx64, id, pattern->seqNumV);
  return true;
}

__attribute__((nonnull)) static void
TgcTx_Burst(TgcTx* ct) {
  struct rte_mbuf* pkts[MaxBurstSize];
  int res = rte_pktmbuf_alloc_bulk(ct->interestMp, pkts, MaxBurstSize);
  if (unlikely(res != 0)) {
    N_LOGW("interestMp-full");
    return;
  }

  TscTime now = rte_get_tsc_cycles();
  for (uint16_t i = 0; i < MaxBurstSize; ++i) {
    while (!likely(TgcTx_MakeInterest(ct, pkts[i], now))) {
    }
  }
  Face_TxBurst(ct->face, (Packet**)pkts, MaxBurstSize);
}

int
TgcTx_Run(TgcTx* ct) {
  TscTime nextTxBurst = rte_get_tsc_cycles();
  int sent = 0;
  while (ThreadCtrl_Continue(ct->ctrl, sent)) {
    if (rte_get_tsc_cycles() < nextTxBurst) {
      continue;
    }
    TgcTx_Burst(ct);
    sent = MaxBurstSize;
    nextTxBurst += ct->burstInterval;
  }
  return 0;
}
