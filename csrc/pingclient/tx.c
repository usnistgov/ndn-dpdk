#include "tx.h"

#include "../core/logger.h"
#include "token.h"

INIT_ZF_LOG(PingClient);

static PingPatternId
PingClientTx_SelectPattern(PingClientTx* ct)
{
  uint32_t rnd = pcg32_random_r(&ct->trafficRng);
  return ct->weight[rnd % ct->nWeights];
}

static void
PingClientTx_MakeInterest(PingClientTx* ct, Packet* npkt, PingTime now)
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

  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  LName nameSuffix = { .length = PINGCLIENT_SUFFIX_LEN,
                       .value = &pattern->seqNum.compT };
  EncodeInterest(pkt, &pattern->tpl, nameSuffix, NonceGen_Next(&ct->nonceGen));

  Packet_SetL3PktType(npkt, L3PktTypeInterest); // for stats; no PInterest*
  Packet_InitLpL3Hdr(npkt)->pitToken =
    PingToken_New(patternId, ct->runNum, now);
  ZF_LOGD("<I pattern=%" PRIu8 " seq=%" PRIx64 "", patternId, seqNum);
}

static void
PingClientTx_Burst(PingClientTx* ct)
{
  Packet* npkts[PINGCLIENT_TX_BURST_SIZE];
  int res = rte_pktmbuf_alloc_bulk(
    ct->interestMp, (struct rte_mbuf**)npkts, PINGCLIENT_TX_BURST_SIZE);
  if (unlikely(res != 0)) {
    ZF_LOGW("interestMp-full");
    return;
  }

  PingTime now = PingTime_Now();
  for (uint16_t i = 0; i < PINGCLIENT_TX_BURST_SIZE; ++i) {
    PingClientTx_MakeInterest(ct, npkts[i], now);
  }
  Face_TxBurst(ct->face, npkts, PINGCLIENT_TX_BURST_SIZE);
}

void
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
}
