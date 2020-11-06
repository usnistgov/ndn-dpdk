#include "face.h"
#include "../core/logger.h"

INIT_ZF_LOG(EthFace);

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

struct rte_flow*
EthFace_SetupFlow(EthFacePriv* priv, const EthLocator* loc, struct rte_flow_error* error)
{
  struct rte_flow_attr attr = {
    .group = 0,
    .priority = 1,
    .ingress = true,
  };

  EthFlowPattern pattern;
  EthFlowPattern_Prepare(&pattern, loc);

  struct rte_flow_action_queue queue = { .index = priv->rxQueue };
  struct rte_flow_action actions[] = {
    { .type = RTE_FLOW_ACTION_TYPE_QUEUE, .conf = &queue },
    { .type = RTE_FLOW_ACTION_TYPE_END },
  };

  struct rte_flow* flow = rte_flow_create(priv->port, &attr, pattern.pattern, actions, error);
  if (flow == NULL) {
    error->cause = (void*)RTE_PTR_DIFF(error->cause, &pattern);
  }
  return flow;
}

uint16_t
EthFace_FlowRxBurst(RxGroup* flowRxg, struct rte_mbuf** pkts, uint16_t nPkts)
{
  EthFacePriv* priv = container_of(flowRxg, EthFacePriv, flowRxg);
  uint16_t nRx = rte_eth_rx_burst(priv->port, priv->rxQueue, pkts, nPkts);
  uint64_t now = rte_get_tsc_cycles();
  for (uint16_t i = 0; i < nRx; ++i) {
    struct rte_mbuf* frame = pkts[i];
    frame->port = priv->faceID;
    frame->timestamp = now;
    rte_pktmbuf_adj(frame, priv->txHdr.len);
    // use txHdr.len instead of rxMatch.len to touch fewer cachelines
    // they are equal, with assertion in ethface.New~iface.NewParams.Init function
  }
  return nRx;
}
