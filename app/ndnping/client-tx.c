#include "client-tx.h"

#include "../../core/logger.h"
#include "token.h"

INIT_ZF_LOG(PingClient);

typedef struct PingClientTxTraffic
{
  uint8_t patternId;
} PingClientTxTraffic;

static PingClientTxTraffic
PingClientTx_SelectTraffic(PingClientTx* ct)
{
  uint32_t rnd = pcg32_random_r(&ct->trafficRng);
  // 32-bit random number
  // 16 bits select a pattern
  // 16 bits unused

  PingClientTxTraffic traffic = { 0 };
  traffic.patternId = (rnd >> 16) % ct->nPatterns;
  return traffic;
}

static void
PingClientTx_MakeInterest(PingClientTx* ct, Packet* npkt, uint64_t now)
{
  PingClientTxTraffic traffic = PingClientTx_SelectTraffic(ct);
  PingClientTxPattern* pattern = &ct->pattern[traffic.patternId];
  ++pattern->nInterests;
  uint64_t seqNum = ++pattern->seqNum.compV;

  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  pkt->data_off = ct->interestMbufHeadroom;
  LName nameSuffix = { .length = PINGCLIENT_SUFFIX_LEN,
                       .value = &pattern->seqNum.compT };
  EncodeInterest(pkt,
                 &pattern->tpl,
                 pattern->tplPrepareBuffer,
                 nameSuffix,
                 NonceGen_Next(&ct->nonceGen),
                 0,
                 NULL);

  Packet_SetL3PktType(npkt, L3PktType_Interest); // for stats; no PInterest*
  Packet_InitLpL3Hdr(npkt)->pitToken =
    PingToken_New(traffic.patternId, ct->runNum, now);
  ZF_LOGD("<I pattern=%" PRIu8 " seq=%" PRIx64 "", traffic.patternId, seqNum);
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

  uint64_t now = Ping_Now();
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
