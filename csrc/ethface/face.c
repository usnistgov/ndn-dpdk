#include "face.h"
#include "../core/logger.h"
#include "../iface/face.h"

N_LOG_INIT(EthFace);

enum {
  EthMaxBurstSize = RTE_DIM(((RxGroupBurstCtx*)NULL)->pkts),

  // https://bugs.dpdk.org/show_bug.cgi?id=1273
  MemifMaxBurstSize = 32,
};

__attribute__((nonnull)) static __rte_always_inline void
EthRxFlow_RxBurst(RxGroup* rxg, RxGroupBurstCtx* ctx, bool skipCheck) {
  EthRxFlow* rxf = container_of(rxg, EthRxFlow, base);
  ctx->nRx = rte_eth_rx_burst(rxf->port, rxf->queue, ctx->pkts, rxf->burstSize);
  uint64_t now = rte_get_tsc_cycles();

  for (uint16_t i = 0; i < ctx->nRx; ++i) {
    struct rte_mbuf* m = ctx->pkts[i];
    if (skipCheck || likely(EthRxMatch_Match(rxf->rxMatch, m))) {
      Mbuf_SetTimestamp(m, now);
      m->port = rxf->faceID;
      rte_pktmbuf_adj(m, rxf->hdrLen);
    } else {
      RxGroupBurstCtx_Drop(ctx, i);
    }
  }
}

__attribute__((nonnull)) static void
EthRxFlow_RxBurst_Unchecked(RxGroup* rxg, RxGroupBurstCtx* ctx) {
  EthRxFlow_RxBurst(rxg, ctx, true);
}

__attribute__((nonnull)) static void
EthRxFlow_RxBurst_Checked(RxGroup* rxg, RxGroupBurstCtx* ctx) {
  EthRxFlow_RxBurst(rxg, ctx, false);
}

struct rte_flow*
EthFace_SetupFlow(EthFacePriv* priv, const uint16_t queues[], int nQueues, const EthLocator* loc,
                  bool isolated, struct rte_flow_error* error) {
  EthLocatorClass c = EthLocator_Classify(loc);
  NDNDPDK_ASSERT(nQueues > 0 && nQueues <= (int)RTE_DIM(priv->rxf));

  struct rte_flow_attr attr = {.ingress = true};

  EthFlowPattern pattern;
  EthFlowPattern_Prepare(&pattern, loc);

  struct rte_flow_action_queue queue = {.index = queues[0]};
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
    {.type = RTE_FLOW_ACTION_TYPE_END},
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
    if (i >= nQueues) {
      *rxf = (const EthRxFlow){0};
      continue;
    }
    *rxf = (const EthRxFlow){
      .base =
        {
          .rxBurst = isolated ? EthRxFlow_RxBurst_Unchecked : EthRxFlow_RxBurst_Checked,
          .rxThread = i,
        },
      .faceID = priv->faceID,
      .port = priv->port,
      .queue = queues[i],
      .burstSize = EthMaxBurstSize,
      .hdrLen = priv->rxMatch.len,
      .rxMatch = isolated ? NULL : &priv->rxMatch,
    };
  }
  return flow;
}

__attribute__((nonnull)) void
EthFace_SetupRxMemif(EthFacePriv* priv, const EthLocator* loc) {
  priv->rxf[0] = (const EthRxFlow){
    .base = {.rxBurst = EthRxFlow_RxBurst_Unchecked, .rxThread = 0},
    .faceID = priv->faceID,
    .port = priv->port,
    .queue = 0,
    .burstSize = RTE_MIN(EthMaxBurstSize, MemifMaxBurstSize),
    .hdrLen = 0,
    .rxMatch = NULL,
  };
}

uint16_t
EthFace_TxBurst(Face* face, struct rte_mbuf** pkts, uint16_t nPkts) {
  EthFacePriv* priv = Face_GetPriv(face);
  for (uint16_t i = 0; i < nPkts; ++i) {
    struct rte_mbuf* m = pkts[i];
    NDNDPDK_ASSERT(!face->txAlign.linearize || rte_pktmbuf_is_contiguous(m));
    EthTxHdr_Prepend(&priv->txHdr, m, i == 0);
  }
  return rte_eth_tx_burst(priv->port, 0, pkts, nPkts);
}

STATIC_ASSERT_FUNC_TYPE(Face_TxBurstFunc, EthFace_TxBurst);
