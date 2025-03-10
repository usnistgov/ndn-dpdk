#include "face.h"
#include "../core/logger.h"
#include "../iface/face.h"
#include "flowdef.h"

N_LOG_INIT(EthFace);

typedef __attribute__((nonnull)) bool (*AcceptFunc)(EthRxFlow*, struct rte_mbuf*);

/**
 * @brief EthRxFlow RX burst function template.
 * @param acceptPkt Packet match function.
 * @param mayHaveUnmatched If true, @c PdumpEthPortUnmatchedCtx is used to trace unmatched packets.
 */
__attribute__((nonnull)) static __rte_always_inline void
EthRxFlow_RxBurst(RxGroup* rxg, RxGroupBurstCtx* ctx, AcceptFunc acceptPkt, bool mayHaveUnmatched) {
  EthRxFlow* rxf = container_of(rxg, EthRxFlow, base);
  ctx->nRx = rte_eth_rx_burst(rxf->port, rxf->queue, ctx->pkts, RTE_DIM(ctx->pkts));
  uint64_t now = rte_get_tsc_cycles();

  PdumpEthPortUnmatchedCtx unmatch;
  if (mayHaveUnmatched) {
    // RCU lock is inherited from RxLoop_Run
    PdumpEthPortUnmatchedCtx_Init(&unmatch, rxf->port);
  }

  for (uint16_t i = 0; i < ctx->nRx; ++i) {
    struct rte_mbuf* m = ctx->pkts[i];
    Mbuf_SetTimestamp(m, now);
    if (likely(acceptPkt(rxf, m))) {
      m->port = rxf->faceID;
      rte_pktmbuf_adj(m, rxf->hdrLen);
    } else {
      RxGroupBurstCtx_Drop(ctx, i);
      if (mayHaveUnmatched && PdumpEthPortUnmatchedCtx_Append(&unmatch, m)) {
        ctx->pkts[i] = NULL;
      }
    }
  }

  if (mayHaveUnmatched) {
    PdumpEthPortUnmatchedCtx_Process(&unmatch);
  }
}

__attribute__((nonnull)) static __rte_always_inline bool
AcceptBypass(__rte_unused EthRxFlow* rxf, __rte_unused struct rte_mbuf* m) {
  return true;
}

/** @brief RX burst function that does not check packets and assumes every packet is matching. */
__attribute__((nonnull)) static void
EthRxFlow_RxBurst_CheckBypass(RxGroup* rxg, RxGroupBurstCtx* ctx) {
  EthRxFlow_RxBurst(rxg, ctx, AcceptBypass, false);
}

__attribute__((nonnull)) static __rte_always_inline bool
AcceptOffload(EthRxFlow* rxf, struct rte_mbuf* m) {
  return EthFace_RxMbufFaceID(m) == rxf->faceID;
}

/** @brief RX burst function that checks FaceID from MARK action. */
__attribute__((nonnull)) static void
EthRxFlow_RxBurst_Isolated_CheckOffload(RxGroup* rxg, RxGroupBurstCtx* ctx) {
  EthRxFlow_RxBurst(rxg, ctx, AcceptOffload, false);
}

/** @brief RX burst function that checks FaceID from MARK action, tracing unmatched packets. */
__attribute__((nonnull)) static void
EthRxFlow_RxBurst_Unisolated_CheckOffload(RxGroup* rxg, RxGroupBurstCtx* ctx) {
  EthRxFlow_RxBurst(rxg, ctx, AcceptOffload, true);
}

__attribute__((nonnull)) static __rte_always_inline bool
AcceptFull(EthRxFlow* rxf, struct rte_mbuf* m) {
  EthFacePriv* priv =
    RTE_PTR_SUB(rxf, offsetof(EthFacePriv, rxf) + sizeof(*rxf) * rxf->base.rxThread);
  return EthRxMatch_Match(&priv->rxMatch, m);
}

/** @brief RX burst function that performs full checks. */
__attribute__((nonnull)) static void
EthRxFlow_RxBurst_CheckFull(RxGroup* rxg, RxGroupBurstCtx* ctx) {
  EthRxFlow_RxBurst(rxg, ctx, AcceptFull, true);
}

struct rte_flow*
EthFace_SetupFlow(EthFacePriv* priv, const uint16_t queues[], int nQueues, const EthLocator* loc,
                  EthFlowFlags flowFlags, struct rte_flow_error* error) {
  EthFlowDef def;
  EthFlowDef_Prepare(&def, loc, &flowFlags, priv->faceID, queues, nQueues);

  struct rte_flow* flow = rte_flow_create(priv->port, &def.attr, def.pattern, def.actions, error);
  if (flow == NULL) {
    EthFlowDef_UpdateError(&def, error);
    return NULL;
  }

  RxGroup_RxBurstFunc rxBurst = EthRxFlow_RxBurst_CheckFull;
  if (flowFlags & EthFlowFlagsMarked) {
    if (flowFlags & EthFlowFlagsIsolated) {
      rxBurst = EthRxFlow_RxBurst_Isolated_CheckOffload;
    } else {
      rxBurst = EthRxFlow_RxBurst_Unisolated_CheckOffload;
    }
  }

  for (int i = 0; i < (int)RTE_DIM(priv->rxf); ++i) {
    EthRxFlow* rxf = &priv->rxf[i];
    if (i >= nQueues) {
      *rxf = (const EthRxFlow){0};
      continue;
    }
    *rxf = (const EthRxFlow){
      .base = {.rxBurst = rxBurst, .rxThread = i},
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
    .base = {.rxBurst = EthRxFlow_RxBurst_CheckBypass, .rxThread = 0},
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
