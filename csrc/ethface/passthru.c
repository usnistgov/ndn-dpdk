#include "passthru.h"
#include "../iface/txloop.h"
#include "face.h"
#include "gtpip.h"

void
EthPassthru_FaceRxInput(Face* face, int rxThread, FaceRxInputCtx* ctx) {
  FaceRxThread* rxt = &face->impl->rx[rxThread];
  EthFacePriv* priv = Face_GetPriv(face);
  EthPassthru* pt = &priv->passthru;

  for (uint16_t i = 0; i < ctx->count; ++i) {
    struct rte_mbuf* pkt = ctx->pkts[i];
    rxt->nFrames[FaceRxThread_cntNOctets] += pkt->pkt_len;
    if (pt->gtpip != NULL && EthGtpip_ProcessUplink(pt->gtpip, pkt)) {
      ++rxt->nFrames[EthPassthru_cntNGtpip];
    } else {
      ++rxt->nFrames[EthPassthru_cntNPkts];
    }
  }

  uint16_t nSent = 0;
  uint16_t tapPort = pt->tapPort;
  if (likely(tapPort != UINT16_MAX)) {
    nSent = rte_eth_tx_burst(pt->tapPort, 0, ctx->pkts, ctx->count);
  }
  ctx->nFree = ctx->count - nSent;
  if (unlikely(ctx->nFree > 0)) {
    rte_memcpy(ctx->frees, ctx->pkts + nSent, sizeof(ctx->pkts[0]) * ctx->nFree);
  }
}

STATIC_ASSERT_FUNC_TYPE(Face_RxInputFunc, EthPassthru_FaceRxInput);

void
EthPassthru_TapPortRxBurst(RxGroup* rxg, RxGroupBurstCtx* ctx) {
  EthPassthru* pt = container_of(rxg, EthPassthru, base);
  EthFacePriv* priv = container_of(pt, EthFacePriv, passthru);
  ctx->nRx = rte_eth_rx_burst(pt->tapPort, 0, ctx->pkts, RTE_DIM(ctx->pkts));
  if (ctx->nRx == 0) {
    return;
  }

  uint64_t now = rte_get_tsc_cycles();
  for (uint16_t i = 0; i < ctx->nRx; ++i) {
    struct rte_mbuf* m = ctx->pkts[i];
    m->port = priv->faceID;
    Mbuf_SetTimestamp(m, now);
  }

  Face_TxBurst(priv->faceID, (Packet**)ctx->pkts, ctx->nRx);

  // RxLoop.ctrl needs non-zero ctx->nRx to properly track empty polls vs valid polls.
  // ctx->dropBits must be set so that RxLoop_Transfer does not dispatch packets as NDN.
  // ctx->pkts must be cleared so that RxLoop_Transfer does not attempt to free the mbufs.
  rte_bitset_set_all(ctx->dropBits, MaxBurstSize);
  memset(ctx->pkts, 0, sizeof(ctx->pkts));
}

STATIC_ASSERT_FUNC_TYPE(RxGroup_RxBurstFunc, EthPassthru_TapPortRxBurst);

uint16_t
EthPassthru_TxLoop(Face* face, int txThread) {
  FaceTxThread* txt = &face->impl->tx[txThread];
  struct rte_mbuf* pkts[MaxBurstSize];
  uint16_t count = rte_ring_dequeue_burst(face->outputQueue, (void**)pkts, MaxBurstSize, NULL);
  if (count == 0) {
    return 0;
  }

  EthFacePriv* priv = Face_GetPriv(face);
  EthPassthru* pt = &priv->passthru;
  if (pt->gtpip == NULL) {
    txt->nFrames[EthPassthru_cntNPkts] += count;
  } else {
    uint64_t nGtpip = rte_popcount64(EthGtpip_ProcessDownlinkBulk(pt->gtpip, pkts, count));
    txt->nFrames[EthPassthru_cntNGtpip] += nGtpip;
    txt->nFrames[EthPassthru_cntNPkts] += count - nGtpip;
  }
  TxLoop_TxFrames(face, txThread, pkts, count);
  return count;
}

STATIC_ASSERT_FUNC_TYPE(Face_TxLoopFunc, EthPassthru_TxLoop);
