#include "face.h"
#include "../core/logger.h"

N_LOG_INIT(EthFace);

__attribute__((nonnull)) static uint16_t
EthRxFlow_RxBurst_Unchecked(RxGroup* rxg, struct rte_mbuf** pkts, uint16_t nPkts)
{
  EthRxFlow* rxf = container_of(rxg, EthRxFlow, base);
  uint16_t nRx = rte_eth_rx_burst(rxf->port, rxf->queue, pkts, nPkts);
  uint64_t now = rte_get_tsc_cycles();
  for (uint16_t i = 0; i < nRx; ++i) {
    struct rte_mbuf* m = pkts[i];
    Mbuf_SetTimestamp(m, now);
    m->port = rxf->faceID;
    rte_pktmbuf_adj(m, rxf->hdrLen);
  }
  return nRx;
}

__attribute__((nonnull)) static uint16_t
EthRxFlow_RxBurst_Checked(RxGroup* rxg, struct rte_mbuf** pkts, uint16_t nPkts)
{
  EthRxFlow* rxf = container_of(rxg, EthRxFlow, base);
  uint16_t nInput = rte_eth_rx_burst(rxf->port, rxf->queue, pkts, nPkts);
  uint64_t now = rte_get_tsc_cycles();

  uint16_t nRx = 0, nRej = 0;
  struct rte_mbuf* rejects[MaxBurstSize];
  for (uint16_t i = 0; i < nInput; ++i) {
    struct rte_mbuf* m = pkts[i];
    if (likely(EthRxMatch_Match(rxf->rxMatch, m))) {
      Mbuf_SetTimestamp(m, now);
      m->port = rxf->faceID;
      pkts[nRx++] = m;
    } else {
      rejects[nRej++] = m;
    }
  }

  if (unlikely(nRej > 0)) {
    rte_pktmbuf_free_bulk(rejects, nRej);
  }
  return nRx;
}

struct rte_flow*
EthFace_SetupFlow(EthFacePriv* priv, uint16_t queues[], int nQueues, const EthLocator* loc,
                  bool isolated, struct rte_flow_error* error)
{
  assert(nQueues > 0 && nQueues < (int)RTE_DIM(priv->rxf));

  struct rte_flow_attr attr = { .ingress = true };

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
    ptrdiff_t offset = RTE_PTR_DIFF(error->cause, &pattern);
    if (offset >= 0 && (size_t)offset < sizeof(pattern)) {
      error->cause = (const void*)RTE_PTR_DIFF(error->cause, &pattern);
    }
    return NULL;
  }

  for (int i = 0; i < (int)RTE_DIM(priv->rxf); ++i) {
    EthRxFlow* rxf = &priv->rxf[i];
    *rxf = (const EthRxFlow){ 0 };
    if (i >= nQueues) {
      continue;
    }
    rxf->base.rxBurstOp = isolated ? EthRxFlow_RxBurst_Unchecked : EthRxFlow_RxBurst_Checked;
    rxf->base.rxThread = i;
    rxf->faceID = priv->faceID;
    rxf->port = priv->port;
    rxf->queue = queues[i];
    rxf->hdrLen = priv->rxMatch.len;
    rxf->rxMatch = &priv->rxMatch;
  }
  return flow;
}

// EthFace currently only supports one TX queue, so queue number is hardcoded with this macro.
#define TX_QUEUE_0 0

uint16_t
EthFace_TxBurst(Face* face, struct rte_mbuf** pkts, uint16_t nPkts)
{
  EthFacePriv* priv = Face_GetPriv(face);
  for (uint16_t i = 0; i < nPkts; ++i) {
    struct rte_mbuf* m = pkts[i];
    NDNDPDK_ASSERT(!face->txAlign.linearize || rte_pktmbuf_is_contiguous(m));
    EthTxHdr_Prepend(&priv->txHdr, m, i == 0);
  }
  return rte_eth_tx_burst(priv->port, TX_QUEUE_0, pkts, nPkts);
}
