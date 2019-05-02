#include "client-tx.h"

#include "../../core/logger.h"
#include "token.h"

INIT_ZF_LOG(PingClient);

static void
PingClientTx_MakeInterest(PingClientTx* ct, Packet* npkt, uint64_t now)
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  pkt->data_off = ct->interestMbufHeadroom;
  uint64_t seqNum = ++ct->suffixComponent.compV;
  uint8_t patternId = PINGCLIENT_SELECT_PATTERN(ct, seqNum);
  PingClientTxPattern* pattern = &ct->pattern[patternId];
  ++pattern->nInterests;

  LName nameSuffix = { .length = 10, .value = &ct->suffixComponent.compT };
  EncodeInterest(pkt,
                 &pattern->tpl,
                 pattern->tplPrepareBuffer,
                 nameSuffix,
                 NonceGen_Next(&ct->nonceGen),
                 0,
                 NULL);
  Packet_SetL3PktType(npkt, L3PktType_Interest); // for stats; no PInterest*
  ZF_LOGD("<I seq=%" PRIx64 " pattern=%d", seqNum, patternId);

  Packet_InitLpL3Hdr(npkt)->pitToken =
    PingToken_New(patternId, ct->runNum, now);
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
