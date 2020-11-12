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
EthFace_SetupFlow(EthFacePriv* priv, int index, uint16_t queue, const EthLocator* loc,
                  struct rte_flow_error* error)
{
  assert(index >= 0 && index < (int)RTE_DIM(priv->rxf));
  EthRxFlow* rxf = &priv->rxf[index];
  *rxf = (const EthRxFlow){ 0 };

  struct rte_flow_attr attr = {
    .group = 0,
    .priority = 1,
    .ingress = true,
  };

  EthFlowPattern pattern;
  EthFlowPattern_Prepare(&pattern, loc);

  struct rte_flow_action_queue queueAction = { .index = queue };
  struct rte_flow_action actions[] = {
    { .type = RTE_FLOW_ACTION_TYPE_QUEUE, .conf = &queueAction },
    { .type = RTE_FLOW_ACTION_TYPE_END },
  };

  struct rte_flow* flow = rte_flow_create(priv->port, &attr, pattern.pattern, actions, error);
  if (flow == NULL) {
    error->cause = (void*)RTE_PTR_DIFF(error->cause, &pattern);
    return NULL;
  }

  rxf->base.rxBurstOp = EthRxFlow_RxBurst;
  rxf->base.rxThread = index;
  rxf->faceID = priv->faceID;
  rxf->port = priv->port;
  rxf->queue = queue;
  rxf->hdrLen = priv->rxMatch.len;
  return flow;
}

// EthFace currently only supports one TX queue, so queue number is hardcoded with this macro.
#define TX_QUEUE_0 0

uint16_t
EthFace_TxBurst(Face* face, struct rte_mbuf** pkts, uint16_t nPkts)
{
  EthFacePriv* priv = Face_GetPrivT(face, EthFacePriv);

  for (uint16_t i = 0; i < nPkts; ++i) {
    EthTxHdr_Prepend(&priv->txHdr, pkts[i]);
  }
  return rte_eth_tx_burst(priv->port, TX_QUEUE_0, pkts, nPkts);
}
