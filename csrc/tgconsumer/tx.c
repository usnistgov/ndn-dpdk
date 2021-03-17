#include "tx.h"

#include "../core/logger.h"
#include "token.h"

N_LOG_INIT(TgConsumer);

__attribute__((nonnull)) static PingPatternId
TgConsumerTx_SelectPattern(TgConsumerTx* ct)
{
  uint32_t rnd = pcg32_random_r(&ct->trafficRng);
  return ct->weight[rnd % ct->nWeights];
}

__attribute__((nonnull)) static void
TgConsumerTx_MakeInterest(TgConsumerTx* ct, struct rte_mbuf* pkt, TgTime now)
{
  PingPatternId patternId = TgConsumerTx_SelectPattern(ct);
  TgConsumerTxPattern* pattern = &ct->pattern[patternId];
  ++pattern->nInterests;

  uint64_t seqNum = 0;
  if (unlikely(pattern->seqNumOffset != 0)) {
    TgConsumerTxPattern* basePattern = &ct->pattern[patternId - 1];
    seqNum = basePattern->seqNum.compV - pattern->seqNumOffset;
    if (unlikely(seqNum == pattern->seqNum.compV)) {
      ++seqNum;
    }
    pattern->seqNum.compV = seqNum;
  } else {
    seqNum = ++pattern->seqNum.compV;
  }
  LName nameSuffix = { .length = TGCONSUMER_SUFFIX_LEN, .value = &pattern->seqNum.compT };

  uint32_t nonce = NonceGen_Next(&ct->nonceGen);
  Packet* npkt = InterestTemplate_Encode(&pattern->tpl, pkt, nameSuffix, nonce);
  Packet_GetLpL3Hdr(npkt)->pitToken = TgToken_New(patternId, ct->runNum, now);
  N_LOGD("<I pattern=%" PRIu8 " seq=%" PRIx64 "", patternId, seqNum);
}

__attribute__((nonnull)) static void
TgConsumerTx_Burst(TgConsumerTx* ct)
{
  struct rte_mbuf* pkts[TGCONSUMER_TX_BURST_SIZE];
  int res = rte_pktmbuf_alloc_bulk(ct->interestMp, pkts, TGCONSUMER_TX_BURST_SIZE);
  if (unlikely(res != 0)) {
    N_LOGW("interestMp-full");
    return;
  }

  TgTime now = TgTime_Now();
  for (uint16_t i = 0; i < TGCONSUMER_TX_BURST_SIZE; ++i) {
    TgConsumerTx_MakeInterest(ct, pkts[i], now);
  }
  Face_TxBurst(ct->face, (Packet**)pkts, TGCONSUMER_TX_BURST_SIZE);
}

int
TgConsumerTx_Run(TgConsumerTx* ct)
{
  TscTime nextTxBurst = rte_get_tsc_cycles();
  while (ThreadStopFlag_ShouldContinue(&ct->stop)) {
    if (rte_get_tsc_cycles() < nextTxBurst) {
      rte_pause();
      continue;
    }
    TgConsumerTx_Burst(ct);
    nextTxBurst += ct->burstInterval;
  }
  return 0;
}
