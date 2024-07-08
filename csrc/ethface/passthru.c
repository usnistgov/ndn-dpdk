#include "passthru.h"
#include "../iface/txloop.h"
#include "face.h"

Packet*
EthPassthru_FaceRxInput(Face* face, int rxThread, struct rte_mbuf* pkt) {
  NDNDPDK_ASSERT(pkt->port == face->id);
  FaceRxThread* rxt = &face->impl->rx[rxThread];
  EthFacePriv* priv = Face_GetPriv(face);
  rxt->nFrames[FaceRxThread_cntNOctets] += pkt->pkt_len;
  ++rxt->nFrames[1];
  uint16_t nSent = 0;
  if (likely(priv->tapPort != UINT16_MAX)) {
    nSent = rte_eth_tx_burst(priv->tapPort, 0, &pkt, 1);
  }
  if (nSent == 0) {
    rte_pktmbuf_free(pkt);
  }
  return NULL;
}

STATIC_ASSERT_FUNC_TYPE(Face_RxInputFunc, EthPassthru_FaceRxInput);

void
EthPassthru_TapPortRxBurst(RxGroup* rxg, RxGroupBurstCtx* ctx) {
  EthRxFlow* rxf = container_of(rxg, EthRxFlow, base);
  uint16_t nRx = rte_eth_rx_burst(rxf->port, rxf->queue, ctx->pkts, RTE_DIM(ctx->pkts));
  if (nRx == 0) {
    return;
  }
  uint64_t now = rte_get_tsc_cycles();

  for (uint16_t i = 0; i < nRx; ++i) {
    struct rte_mbuf* m = ctx->pkts[i];
    Mbuf_SetTimestamp(m, now);
  }

  Face_TxBurst(rxf->faceID, (Packet**)ctx->pkts, nRx);
}

STATIC_ASSERT_FUNC_TYPE(RxGroup_RxBurstFunc, EthPassthru_TapPortRxBurst);

uint16_t
EthPassthru_TxLoop(Face* face, int txThread) {
  FaceTxThread* txt = &face->impl->tx[txThread];
  struct rte_mbuf* frames[MaxBurstSize];
  uint16_t count = rte_ring_dequeue_burst(face->outputQueue, (void**)frames, MaxBurstSize, NULL);
  if (count > 0) {
    txt->nFrames[1] += count;
    TxLoop_TxFrames(face, txThread, frames, count);
  }
  return count;
}

STATIC_ASSERT_FUNC_TYPE(Face_TxLoopFunc, EthPassthru_TxLoop);
