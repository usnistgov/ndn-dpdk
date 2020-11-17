#include "face.h"
#include "../core/logger.h"

INIT_ZF_LOG(EthFace);

__attribute__((nonnull)) static uint16_t
EthRxFlow_RxBurst(RxGroup* rxg, struct rte_mbuf** pkts, uint16_t nPkts)
{
  EthRxFlow* rxf = (EthRxFlow*)rxg;
  uint16_t nRx = rte_eth_rx_burst(rxf->port, rxf->queue, pkts, nPkts);
  uint64_t now = rte_get_tsc_cycles();
  for (uint16_t i = 0; i < nRx; ++i) {
    struct rte_mbuf* frame = pkts[i];
    frame->port = rxf->faceID;
    frame->timestamp = now;
    rte_pktmbuf_adj(frame, rxf->hdrLen);
  }
  return nRx;
}

struct rte_flow*
EthFace_SetupFlow(EthFacePriv* priv, uint16_t queues[], int nQueues, const EthLocator* loc,
                  struct rte_flow_error* error)
{
  assert(nQueues > 0 && nQueues < (int)RTE_DIM(priv->rxf));

  struct rte_flow_attr attr = {
    .group = 0,
    .priority = 1,
    .ingress = true,
  };

  EthFlowPattern pattern;
  EthFlowPattern_Prepare(&pattern, loc);

  struct rte_flow_action_queue queue = { .index = queues[0] };
  struct rte_flow_action_rss rss = {
    .level = 1,
    .types = ETH_RSS_NONFRAG_IPV4_UDP,
    .queue = queues,
    .queue_num = nQueues,
  };
  struct rte_flow_action actions[] = {
    {
      .type = nQueues > 1 ? RTE_FLOW_ACTION_TYPE_RSS : RTE_FLOW_ACTION_TYPE_QUEUE,
      .conf = nQueues > 1 ? (const void*)&rss : (const void*)&queue,
    },
    { .type = RTE_FLOW_ACTION_TYPE_END },
  };

  struct rte_flow* flow = rte_flow_create(priv->port, &attr, pattern.pattern, actions, error);
  if (flow == NULL) {
    error->cause = (const void*)RTE_PTR_DIFF(error->cause, &pattern);
    return NULL;
  }

  for (int i = 0; i < (int)RTE_DIM(priv->rxf); ++i) {
    EthRxFlow* rxf = &priv->rxf[i];
    *rxf = (const EthRxFlow){ 0 };
    if (i >= nQueues) {
      continue;
    }
    rxf->base.rxBurstOp = EthRxFlow_RxBurst;
    rxf->base.rxThread = i;
    rxf->faceID = priv->faceID;
    rxf->port = priv->port;
    rxf->queue = queues[i];
    rxf->hdrLen = priv->rxMatch.len;
  }
  return flow;
}

// EthFace currently only supports one TX queue, so queue number is hardcoded with this macro.
#define TX_QUEUE_0 0

uint16_t
EthFace_TxBurst(Face* face, struct rte_mbuf** pkts, uint16_t nPkts)
{
  EthFacePriv* priv = Face_GetPrivT(face, EthFacePriv);

  for (uint16_t i = 0; i < nPkts; ++i) {
    struct rte_mbuf* m = pkts[i];
    EthTxHdr_Prepend(&priv->txHdr, m);
    if (unlikely(priv->txLinearize)) {
      if (rte_pktmbuf_tailroom(m) < m->pkt_len - m->data_len) {
        // TODO cleanup `m->data_off = m->buf_len` in ndni and iface packages to avoid this memmove
        const void* buf = rte_pktmbuf_mtod(m, const void*);
        m->data_off = RTE_PKTMBUF_HEADROOM;
        memmove(rte_pktmbuf_mtod(m, void*), buf, m->data_len);
        m->data_off = RTE_PKTMBUF_HEADROOM;
      }
      int res = rte_pktmbuf_linearize(m);
      NDNDPDK_ASSERT(res == 0);
    }
  }
  return rte_eth_tx_burst(priv->port, TX_QUEUE_0, pkts, nPkts);
}
