#include "passthru.h"
#include "../iface/txloop.h"
#include "face.h"
#include "gtpip.h"

Packet*
EthPassthru_FaceRxInput(Face* face, int rxThread, struct rte_mbuf* pkt) {
  NDNDPDK_ASSERT(pkt->port == face->id);
  FaceRxThread* rxt = &face->impl->rx[rxThread];
  EthFacePriv* priv = Face_GetPriv(face);
  EthPassthru* pt = &priv->passthru;

  rxt->nFrames[FaceRxThread_cntNOctets] += pkt->pkt_len;
  if (pt->gtpip != NULL && EthGtpip_ProcessUplink(pt->gtpip, pkt)) {
    ++rxt->nFrames[EthPassthru_cntNGtpip];
  } else {
    ++rxt->nFrames[EthPassthru_cntNPkts];
  }

  uint16_t nSent = 0;
  uint16_t tapPort = priv->passthru.tapPort;
  if (likely(tapPort != UINT16_MAX)) {
    nSent = rte_eth_tx_burst(tapPort, 0, &pkt, 1);
  }
  if (unlikely(nSent == 0)) {
    rte_pktmbuf_free(pkt);
  }
  return NULL;
}

STATIC_ASSERT_FUNC_TYPE(Face_RxInputFunc, EthPassthru_FaceRxInput);

void
EthPassthru_TapPortRxBurst(RxGroup* rxg, RxGroupBurstCtx* ctx) {
  EthPassthru* pt = container_of(rxg, EthPassthru, base);
  EthFacePriv* priv = container_of(pt, EthFacePriv, passthru);
  uint16_t nRx = rte_eth_rx_burst(pt->tapPort, 0, ctx->pkts, RTE_DIM(ctx->pkts));
  if (nRx == 0) {
    return;
  }

  uint64_t now = rte_get_tsc_cycles();
  for (uint16_t i = 0; i < nRx; ++i) {
    struct rte_mbuf* m = ctx->pkts[i];
    Mbuf_SetTimestamp(m, now);
  }

  Face_TxBurst(priv->faceID, (Packet**)ctx->pkts, nRx);
}

STATIC_ASSERT_FUNC_TYPE(RxGroup_RxBurstFunc, EthPassthru_TapPortRxBurst);

uint16_t
EthPassthru_TxLoop(Face* face, int txThread) {
  FaceTxThread* txt = &face->impl->tx[txThread];
  struct rte_mbuf* frames[MaxBurstSize];
  uint16_t nTx = rte_ring_dequeue_burst(face->outputQueue, (void**)frames, MaxBurstSize, NULL);
  if (nTx == 0) {
    return 0;
  }

  EthFacePriv* priv = Face_GetPriv(face);
  EthPassthru* pt = &priv->passthru;
  for (uint16_t i = 0; i < nTx; ++i) {
    struct rte_mbuf* m = frames[i];
    if (pt->gtpip != NULL && EthGtpip_ProcessDownlink(pt->gtpip, m)) {
      ++txt->nFrames[EthPassthru_cntNGtpip];
    } else {
      ++txt->nFrames[EthPassthru_cntNPkts];
    }
  }
  TxLoop_TxFrames(face, txThread, frames, nTx);
  return nTx;
}

STATIC_ASSERT_FUNC_TYPE(Face_TxLoopFunc, EthPassthru_TxLoop);
