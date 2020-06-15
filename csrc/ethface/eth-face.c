#include "eth-face.h"
#include "../core/logger.h"

uint8_t
EthFaceEtherHdr_Init(EthFaceEtherHdr* hdr,
                     const struct rte_ether_addr* local,
                     const struct rte_ether_addr* remote,
                     uint16_t vlan0,
                     uint16_t vlan1)
{
  hdr->eth.ether_type = rte_cpu_to_be_16(NDN_ETHERTYPE);
  hdr->vlan0.eth_proto = rte_cpu_to_be_16(NDN_ETHERTYPE);
  hdr->vlan1.eth_proto = rte_cpu_to_be_16(NDN_ETHERTYPE);

  rte_ether_addr_copy(remote, &hdr->eth.d_addr);
  rte_ether_addr_copy(local, &hdr->eth.s_addr);
  if (vlan0 == 0) {
    return offsetof(EthFaceEtherHdr, vlan0);
  }

  hdr->vlan0.vlan_tci = rte_cpu_to_be_16(vlan0);
  hdr->eth.ether_type = rte_cpu_to_be_16(RTE_ETHER_TYPE_VLAN);
  if (vlan1 == 0) {
    return offsetof(EthFaceEtherHdr, vlan1);
  }

  hdr->vlan1.vlan_tci = rte_cpu_to_be_16(vlan1);
  hdr->vlan0.eth_proto = rte_cpu_to_be_16(RTE_ETHER_TYPE_VLAN);
  hdr->eth.ether_type = rte_cpu_to_be_16(RTE_ETHER_TYPE_QINQ);
  return sizeof(EthFaceEtherHdr);
}

INIT_ZF_LOG(EthFace);

// EthFace currently only supports one TX queue,
// so queue number is hardcoded with this macro.
#define TX_QUEUE_0 0

uint16_t
EthFace_TxBurst(Face* face, struct rte_mbuf** pkts, uint16_t nPkts)
{
  EthFacePriv* priv = Face_GetPrivT(face, EthFacePriv);

  for (uint16_t i = 0; i < nPkts; ++i) {
    char* room = rte_pktmbuf_prepend(pkts[i], priv->txHdrLen);
    assert(room != NULL); // enough headroom is required
    rte_memcpy(room, &priv->txHdr, priv->txHdrLen);
  }
  return rte_eth_tx_burst(priv->port, TX_QUEUE_0, pkts, nPkts);
}

struct rte_flow*
EthFace_SetupFlow(EthFacePriv* priv, struct rte_flow_error* error)
{
  const EthFaceEtherHdr* hdr = &priv->txHdr;
  struct rte_flow_attr attr = {
    .group = 0,
    .priority = 1,
    .ingress = true,
  };

  struct rte_flow_item_eth ethMask;
  memset(&ethMask, 0xFF, sizeof(ethMask));
  struct rte_flow_item_eth ethSpec = { .type = hdr->eth.ether_type };
  if (rte_is_multicast_ether_addr(&hdr->eth.d_addr)) {
    rte_ether_addr_copy(&hdr->eth.d_addr, &ethSpec.dst);
    memset(&ethMask.src, 0x00, sizeof(ethMask.src));
  } else { // unicast
    rte_ether_addr_copy(&hdr->eth.s_addr, &ethSpec.dst);
    rte_ether_addr_copy(&hdr->eth.d_addr, &ethSpec.src);
  }
  struct rte_flow_item_vlan vlanMask = { .tci = rte_cpu_to_be_16(0x0FFF),
                                         .inner_type = 0xFFFF };
  struct rte_flow_item_vlan vlanSpec0 = { .tci = hdr->vlan0.vlan_tci,
                                          .inner_type = hdr->vlan0.eth_proto };
  struct rte_flow_item_vlan vlanSpec1 = { .tci = hdr->vlan1.vlan_tci,
                                          .inner_type = hdr->vlan1.eth_proto };

  struct rte_flow_item pattern[4] = {
    {
      .type = RTE_FLOW_ITEM_TYPE_ETH,
      .mask = &ethMask,
      .spec = &ethSpec,
    },
    {
      .type = priv->txHdrLen > offsetof(EthFaceEtherHdr, vlan0)
                ? RTE_FLOW_ITEM_TYPE_VLAN
                : RTE_FLOW_ITEM_TYPE_END,
      .mask = &vlanMask,
      .spec = &vlanSpec0,
    },
    {
      .type = priv->txHdrLen > offsetof(EthFaceEtherHdr, vlan1)
                ? RTE_FLOW_ITEM_TYPE_VLAN
                : RTE_FLOW_ITEM_TYPE_END,
      .mask = &vlanMask,
      .spec = &vlanSpec1,
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
    rte_pktmbuf_adj(frame, priv->txHdrLen);
  }
  return nRx;
}
