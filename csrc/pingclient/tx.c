#include "tx.h"

#include "../core/logger.h"
#include "token.h"

INIT_ZF_LOG(PingClient);

__attribute__((nonnull)) static PingPatternId
PingClientTx_SelectPattern(PingClientTx* ct)
{
  uint32_t rnd = pcg32_random_r(&ct->trafficRng);
  return ct->weight[rnd % ct->nWeights];
}

__attribute__((nonnull)) static void
PingClientTx_MakeInterest(PingClientTx* ct, struct rte_mbuf* pkt, PingTime now)
{
  PingPatternId patternId = PingClientTx_SelectPattern(ct);
  PingClientTxPattern* pattern = &ct->pattern[patternId];
  ++pattern->nInterests;

  uint64_t seqNum = 0;
  if (unlikely(pattern->seqNumOffset != 0)) {
    PingClientTxPattern* basePattern = &ct->pattern[patternId - 1];
    seqNum = basePattern->seqNum.compV - pattern->seqNumOffset;
    if (unlikely(seqNum == pattern->seqNum.compV)) {
      ++seqNum;
    }
    pattern->seqNum.compV = seqNum;
  } else {
    seqNum = ++pattern->seqNum.compV;
  }
  LName nameSuffix = { .length = PINGCLIENT_SUFFIX_LEN, .value = &pattern->seqNum.compT };

  uint32_t nonce = NonceGen_Next(&ct->nonceGen);
  Packet* npkt = InterestTemplate_Encode(&pattern->tpl, pkt, nameSuffix, nonce);
  Packet_GetLpL3Hdr(npkt)->pitToken = PingToken_New(patternId, ct->runNum, now);
  ZF_LOGD("<I pattern=%" PRIu8 " seq=%" PRIx64 "", patternId, seqNum);
}

__attribute__((nonnull)) static void
PingClientTx_Burst(PingClientTx* ct)
{
  struct rte_mbuf* pkts[PINGCLIENT_TX_BURST_SIZE];
  int res = rte_pktmbuf_alloc_bulk(ct->interestMp, pkts, PINGCLIENT_TX_BURST_SIZE);
  if (unlikely(res != 0)) {
    ZF_LOGW("interestMp-full");
    return;
  }

  PingTime now = PingTime_Now();
  for (uint16_t i = 0; i < PINGCLIENT_TX_BURST_SIZE; ++i) {
    PingClientTx_MakeInterest(ct, pkts[i], now);
  }
  Face_TxBurst(ct->face, (Packet**)pkts, PINGCLIENT_TX_BURST_SIZE);
}

int
PingClientTx_Run(PingClientTx* ct)
{
  TscTime nextTxBurst = rte_get_tsc_cycles();
  while (ThreadStopFlag_ShouldContinue(&ct->stop)) {
    if (rte_get_tsc_cycles() < nextTxBurst) {
      rte_pause();
      continue;
    }
    PingClientTx_Burst(ct);
    nextTxBurst += ct->burstInterval;
  }
  return 0;
}
