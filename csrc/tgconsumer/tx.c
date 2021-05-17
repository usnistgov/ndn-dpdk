#include "tx.h"

#include "../core/logger.h"

N_LOG_INIT(Tgc);

__attribute__((nonnull)) static __rte_always_inline uint8_t
TgcTx_SelectPattern(TgcTx* ct)
{
  uint32_t w = pcg32_boundedrand_r(&ct->trafficRng, ct->nWeights);
  return ct->weight[w];
}

__attribute__((nonnull)) static void
TgcTx_MakeInterest(TgcTx* ct, struct rte_mbuf* pkt, TscTime now)
{
  uint8_t id = TgcTx_SelectPattern(ct);
  TgcTxPattern* pattern = &ct->pattern[id];
  ++pattern->nInterests;

  uint64_t seqNum = 0;
  if (unlikely(pattern->seqNumOffset != 0)) {
    TgcTxPattern* basePattern = &ct->pattern[id - 1];
    seqNum = basePattern->seqNum.compV - pattern->seqNumOffset;
    if (unlikely(seqNum == pattern->seqNum.compV)) {
      ++seqNum;
    }
    pattern->seqNum.compV = seqNum;
  } else {
    seqNum = ++pattern->seqNum.compV;
  }
  LName nameSuffix = { .length = TGCONSUMER_SEQNUM_SIZE, .value = &pattern->seqNum.compT };

  uint32_t nonce = NonceGen_Next(&ct->nonceGen);
  Packet* npkt = InterestTemplate_Encode(&pattern->tpl, pkt, nameSuffix, nonce);
  TgcToken_Set(&Packet_GetLpL3Hdr(npkt)->pitToken, id, ct->runNum, now);
  N_LOGD("<I pattern=%" PRIu8 " seq=%" PRIx64 "", id, seqNum);
}

__attribute__((nonnull)) static void
TgcTx_Burst(TgcTx* ct)
{
  struct rte_mbuf* pkts[MaxBurstSize];
  int res = rte_pktmbuf_alloc_bulk(ct->interestMp, pkts, MaxBurstSize);
  if (unlikely(res != 0)) {
    N_LOGW("interestMp-full");
    return;
  }

  TscTime now = rte_get_tsc_cycles();
  for (uint16_t i = 0; i < MaxBurstSize; ++i) {
    TgcTx_MakeInterest(ct, pkts[i], now);
  }
  Face_TxBurst(ct->face, (Packet**)pkts, MaxBurstSize);
}

int
TgcTx_Run(TgcTx* ct)
{
  TscTime nextTxBurst = rte_get_tsc_cycles();
  while (ThreadStopFlag_ShouldContinue(&ct->stop)) {
    if (rte_get_tsc_cycles() < nextTxBurst) {
      rte_pause();
      continue;
    }
    TgcTx_Burst(ct);
    nextTxBurst += ct->burstInterval;
  }
  return 0;
}
