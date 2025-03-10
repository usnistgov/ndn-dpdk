#include "face.h"
#include "../core/logger.h"
#include "../iface/face.h"
#include "flowdef.h"

N_LOG_INIT(EthFace);

__attribute__((nonnull)) static __rte_always_inline void
EthRxFlow_RxBurst(RxGroup* rxg, RxGroupBurstCtx* ctx, bool isolated, bool memif) {
  EthRxFlow* rxf = container_of(rxg, EthRxFlow, base);
  ctx->nRx = rte_eth_rx_burst(rxf->port, rxf->queue, ctx->pkts, RTE_DIM(ctx->pkts));
  uint64_t now = rte_get_tsc_cycles();

  PdumpEthPortUnmatchedCtx unmatch;
  if (isolated) {
    POISON(&unmatch);
  } else {
    // RCU lock is inherited from RxLoop_Run
    PdumpEthPortUnmatchedCtx_Init(&unmatch, rxf->port);
  }

  for (uint16_t i = 0; i < ctx->nRx; ++i) {
    struct rte_mbuf* m = ctx->pkts[i];
    Mbuf_SetTimestamp(m, now);
    if (memif || likely(EthFace_RxMbufFaceID(m) == rxf->faceID)) {
      m->port = rxf->faceID;
      rte_pktmbuf_adj(m, rxf->hdrLen);
    } else {
      RxGroupBurstCtx_Drop(ctx, i);
      if (!isolated && PdumpEthPortUnmatchedCtx_Append(&unmatch, m)) {
        ctx->pkts[i] = NULL;
      }
    }
  }

  if (!isolated) {
    PdumpEthPortUnmatchedCtx_Process(&unmatch);
  }
}

__attribute__((nonnull)) static void
EthRxFlow_RxBurst_Memif(RxGroup* rxg, RxGroupBurstCtx* ctx) {
  EthRxFlow_RxBurst(rxg, ctx, true, true);
}

__attribute__((nonnull)) static void
EthRxFlow_RxBurst_Isolated(RxGroup* rxg, RxGroupBurstCtx* ctx) {
  EthRxFlow_RxBurst(rxg, ctx, true, false);
}

__attribute__((nonnull)) static void
EthRxFlow_RxBurst_Checked(RxGroup* rxg, RxGroupBurstCtx* ctx) {
  EthRxFlow_RxBurst(rxg, ctx, false, false);
}

struct rte_flow*
EthFace_SetupFlow(EthFacePriv* priv, const uint16_t queues[], int nQueues, const EthLocator* loc,
                  bool isolated, EthFlowFlags flowFlags, struct rte_flow_error* error) {
  EthFlowDef def;
  EthFlowDef_Prepare(&def, loc, flowFlags, priv->faceID, queues, nQueues);

  struct rte_flow* flow = rte_flow_create(priv->port, &def.attr, def.pattern, def.actions, error);
  if (flow == NULL) {
    EthFlowDef_UpdateError(&def, error);
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
          .rxBurst = isolated ? EthRxFlow_RxBurst_Isolated : EthRxFlow_RxBurst_Checked,
          .rxThread = i,
        },
      .faceID = priv->faceID,
      .port = priv->port,
      .queue = queues[i],
      .hdrLen = priv->rxMatch.len,
    };
  }
  return flow;
}

void
EthFace_SetupRxMemif(EthFacePriv* priv, const EthLocator* loc) {
  priv->rxf[0] = (const EthRxFlow){
    .base = {.rxBurst = EthRxFlow_RxBurst_Memif, .rxThread = 0},
    .faceID = priv->faceID,
    .port = priv->port,
    .queue = 0,
    .hdrLen = 0,
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
