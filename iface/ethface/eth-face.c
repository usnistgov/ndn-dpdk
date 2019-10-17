#include "eth-face.h"
#include "../../core/logger.h"

INIT_ZF_LOG(EthFace);

// EthFace currently only supports one TX queue,
// so queue number is hardcoded with this macro.
#define TX_QUEUE_0 0

uint16_t
EthFace_TxBurst(Face* face, struct rte_mbuf** pkts, uint16_t nPkts)
{
  EthFacePriv* priv = Face_GetPrivT(face, EthFacePriv);

  for (uint16_t i = 0; i < nPkts; ++i) {
    char* room = rte_pktmbuf_prepend(pkts[i], sizeof(priv->txHdr));
    assert(room != NULL); // enough headroom is required
    rte_memcpy(room, &priv->txHdr, sizeof(priv->txHdr));

    // XXX (1) FaceImpl_CountSent only wants transmitted packets, not tail-dropped packets.
    // XXX (2) FaceImpl_CountSent should be invoked after transmission, not before enqueuing.
    // XXX note: invoking FaceImpl_CountSent from txCallback has the same problems.
    FaceImpl_CountSent(face, pkts[i]);
  }
  return rte_eth_tx_burst(priv->port, TX_QUEUE_0, pkts, nPkts);
}

struct rte_flow*
EthFace_SetupFlow(EthFacePriv* priv, struct rte_flow_error* error)
{
  struct rte_flow_attr attr = {
    .group = 0,
    .priority = 1,
    .ingress = true,
  };

  struct rte_flow_item_eth ethMask = { .type = 0xFFFF };
  struct rte_flow_item_eth ethSpec = { .type = priv->txHdr.ether_type };
  if ((priv->txHdr.d_addr.addr_bytes[0] & 0x01) != 0) { // multicast
    memset(&ethMask.dst, 0xFF, RTE_ETHER_ADDR_LEN);
    memcpy(&ethSpec.dst, &priv->txHdr.d_addr, RTE_ETHER_ADDR_LEN);
  } else { // unicast
    memset(&ethMask.src, 0xFF, RTE_ETHER_ADDR_LEN);
    memcpy(&ethSpec.src, &priv->txHdr.d_addr, RTE_ETHER_ADDR_LEN);
  }

  struct rte_flow_item pattern[2] = {
    {
      .type = RTE_FLOW_ITEM_TYPE_ETH,
      .mask = &ethMask,
      .spec = &ethSpec,
    },
    {
      .type = RTE_FLOW_ITEM_TYPE_END,
    },
  };

  struct rte_flow_action_queue queue = { .index = priv->rxQueue };

  struct rte_flow_action actions[2] = {
    { .type = RTE_FLOW_ACTION_TYPE_QUEUE, .conf = &queue },
    {
      .type = RTE_FLOW_ACTION_TYPE_END,
    },
  };

  return rte_flow_create(priv->port, &attr, pattern, actions, error);
}

uint16_t
EthFace_FlowRxBurst(RxGroup* flowRxg, struct rte_mbuf** pkts, uint16_t nPkts)
{
  EthFacePriv* priv = container_of(flowRxg, EthFacePriv, flowRxg);
  uint16_t nRx = rte_eth_rx_burst(priv->port, priv->rxQueue, pkts, nPkts);
  uint64_t now = rte_get_tsc_cycles();
  for (uint16_t i = 0; i < nRx; ++i) {
    struct rte_mbuf* frame = pkts[i];
    frame->port = priv->faceId;
    // TODO offload timestamping to hardware where available
    frame->timestamp = now;
    rte_pktmbuf_adj(frame, sizeof(struct rte_ether_hdr));
  }
  return nRx;
}
