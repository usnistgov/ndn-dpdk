#include "passthru.h"
#include "../iface/txloop.h"
#include "face.h"
#include "gtpip.h"

__attribute__((nonnull)) static inline void
EthPassthru_FaceRxInput_Gtpip(FaceRxThread* rxt, EthGtpip* g, FaceRxInputCtx* ctx, uint16_t* nPkts,
                              uint16_t* nTx) {
  uint64_t gtpipMask = EthGtpip_ProcessUplinkBulk(g, ctx->pkts, ctx->count);
  if (gtpipMask == 0) {
    // no GTP-IP packets
    return;
  }

  uint16_t nGtpip = rte_popcount64(gtpipMask);
  rxt->nFrames[EthPassthru_cntNGtpip] += nGtpip;
  *nPkts -= nGtpip;
  if (g->n6Face == 0) {
    // GTP-IP packets are interleaved with other packets and will be sent via TAP netif
    return;
  }

  // GTP-IP packets must be separated from other packets and will be sent via N6 face
  for (uint16_t i = 0, jPkt = 0, jGtpip = 0; i < ctx->count; ++i) {
    struct rte_mbuf* pkt = ctx->pkts[i];
    if (rte_bit_test(&gtpipMask, i)) {
      rte_memcpy(rte_pktmbuf_mtod(pkt, uint8_t*), g->n6Mac, sizeof(g->n6Mac));
      ctx->npkts[jGtpip++] = (Packet*)pkt;
    } else {
      ctx->pkts[jPkt++] = pkt;
    }
  }
  Face_TxBurst(g->n6Face, ctx->npkts, nGtpip);
  *nTx = *nPkts;
}

void
EthPassthru_FaceRxInput(Face* face, int rxThread, FaceRxInputCtx* ctx) {
  FaceRxThread* rxt = &face->impl->rx[rxThread];
  EthFacePriv* priv = Face_GetPriv(face);
  EthPassthru* pt = &priv->passthru;

  for (uint16_t i = 0; i < ctx->count; ++i) {
    struct rte_mbuf* pkt = ctx->pkts[i];
    rxt->nFrames[FaceRxThread_cntNOctets] += pkt->pkt_len;
  }

  uint16_t nPkts = ctx->count; // how many packets to be counted in the nPkts counter
  uint16_t nTx = ctx->count;   // how many packets to be transmitted via TAP netif
  if (pt->gtpip != NULL) {
    EthPassthru_FaceRxInput_Gtpip(rxt, pt->gtpip, ctx, &nPkts, &nTx);
  }
  rxt->nFrames[EthPassthru_cntNPkts] += nPkts;

  if (pt->n3Face != 0) {
    Face_TxBurst(pt->n3Face, (Packet**)ctx->pkts, nTx);
    return;
  }

  uint16_t nSent = 0;
  uint16_t tapPort = pt->tapPort;
  if (likely(tapPort != UINT16_MAX) && nTx > 0) {
    nSent = rte_eth_tx_burst(pt->tapPort, 0, ctx->pkts, nTx);
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
