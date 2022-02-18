#include "face.h"
#include "../core/logger.h"

N_LOG_INIT(EthFace);

__attribute__((nonnull)) static void
EthRxFlow_RxBurst_Unchecked(RxGroup* rxg, RxGroupBurstCtx* ctx)
{
  EthRxFlow* rxf = container_of(rxg, EthRxFlow, base);
  ctx->nRx = rte_eth_rx_burst(rxf->port, rxf->queue, ctx->pkts, RTE_DIM(ctx->pkts));
  uint64_t now = rte_get_tsc_cycles();
  for (uint16_t i = 0; i < ctx->nRx; ++i) {
    struct rte_mbuf* m = ctx->pkts[i];
    Mbuf_SetTimestamp(m, now);
    m->port = rxf->faceID;
    rte_pktmbuf_adj(m, rxf->hdrLen);
  }
}

__attribute__((nonnull)) static void
EthRxFlow_RxBurst_Checked(RxGroup* rxg, RxGroupBurstCtx* ctx)
{
  EthRxFlow* rxf = container_of(rxg, EthRxFlow, base);
  ctx->nRx = rte_eth_rx_burst(rxf->port, rxf->queue, ctx->pkts, RTE_DIM(ctx->pkts));
  uint64_t now = rte_get_tsc_cycles();

  for (uint16_t i = 0; i < ctx->nRx; ++i) {
    struct rte_mbuf* m = ctx->pkts[i];
    if (likely(EthRxMatch_Match(rxf->rxMatch, m))) {
      Mbuf_SetTimestamp(m, now);
      m->port = rxf->faceID;
    } else {
      RxGroupBurstCtx_Drop(ctx, i);
    }
  }
}

struct rte_flow*
EthFace_SetupFlow(EthFacePriv* priv, uint16_t queues[], int nQueues, const EthLocator* loc,
                  bool isolated, struct rte_flow_error* error)
{
  EthLocatorClass c = EthLocator_Classify(loc);
  NDNDPDK_ASSERT(nQueues > 0 && nQueues <= (int)RTE_DIM(priv->rxf));

  struct rte_flow_attr attr = { .ingress = true };

  EthFlowPattern pattern;
  EthFlowPattern_Prepare(&pattern, loc);

  struct rte_flow_action_queue queue = { .index = queues[0] };
  struct rte_flow_action_rss rss = {
    .level = 1,
    .types = c.v4 ? RTE_ETH_RSS_NONFRAG_IPV4_UDP : RTE_ETH_RSS_NONFRAG_IPV6_UDP,
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
      error->cause = (const void*)offset;
    }
    return NULL;
  }

  for (int i = 0; i < (int)RTE_DIM(priv->rxf); ++i) {
    EthRxFlow* rxf = &priv->rxf[i];
    *rxf = (const EthRxFlow){ 0 };
    if (i >= nQueues) {
      continue;
    }
    rxf->base.rxBurst = isolated ? EthRxFlow_RxBurst_Unchecked : EthRxFlow_RxBurst_Checked;
    rxf->base.rxThread = i;
    rxf->faceID = priv->faceID;
    rxf->port = priv->port;
    rxf->queue = queues[i];
    rxf->hdrLen = priv->rxMatch.len;
    rxf->rxMatch = &priv->rxMatch;
  }
  return flow;
}

__attribute__((nonnull)) void
EthFace_SetupRxMemif(EthFacePriv* priv, const EthLocator* loc)
{
  priv->rxf[0] = (const EthRxFlow){
    .base = {
      .rxBurst = EthRxFlow_RxBurst_Unchecked,
      .rxThread = 0,
    },
    .faceID = priv->faceID,
    .port = priv->port,
    .queue = 0,
    .hdrLen = 0,
    .rxMatch = NULL,
  };
}

uint16_t
EthFace_TxBurst(Face* face, struct rte_mbuf** pkts, uint16_t nPkts)
{
  EthFacePriv* priv = Face_GetPriv(face);
  for (uint16_t i = 0; i < nPkts; ++i) {
    struct rte_mbuf* m = pkts[i];
    NDNDPDK_ASSERT(!face->txAlign.linearize || rte_pktmbuf_is_contiguous(m));
    EthTxHdr_Prepend(&priv->txHdr, m, i == 0);
  }
  return rte_eth_tx_burst(priv->port, 0, pkts, nPkts);
}
