#include "rxgroup.h"
#include <rte_ethdev.h>

static bool
EthRxTable_Accept(EthRxTable* rxt, struct rte_mbuf* frame, uint64_t now)
{
  assert(frame->data_len >= sizeof(struct ether_hdr));
  const struct ether_hdr* eth =
    rte_pktmbuf_mtod(frame, const struct ether_hdr*);

  if (unlikely(eth->ether_type != rte_cpu_to_be_16(NDN_ETHERTYPE))) {
    rte_pktmbuf_free(frame);
    return false;
  }

  bool isMulticast = eth->d_addr.addr_bytes[0] & 0x01;
  if (isMulticast) {
    frame->port = rxt->multicast;
  } else {
    uint8_t srcLastOctet = eth->s_addr.addr_bytes[5];
    frame->port = rxt->unicast[srcLastOctet];
  }

  frame->timestamp = now;

  rte_pktmbuf_adj(frame, sizeof(struct ether_hdr));
  return true;
}

uint16_t
EthRxTable_RxBurst(RxGroup* rxg, struct rte_mbuf** pkts, uint16_t nPkts)
{
  EthRxTable* rxt = (EthRxTable*)rxg;
  uint16_t nInput = rte_eth_rx_burst(rxt->port, rxt->queue, pkts, nPkts);
  uint64_t now = rte_get_tsc_cycles();
  uint16_t nRx = 0;
  for (uint16_t i = 0; i < nInput; ++i) {
    struct rte_mbuf* frame = pkts[i];
    if (likely(EthRxTable_Accept(rxt, frame, now))) {
      pkts[nRx++] = frame;
    }
  }
  return nRx;
}

bool
EthRxFlow_Setup(EthRxFlow* rxf,
                struct ether_addr* sender,
                struct rte_flow_error* error)
{
  struct rte_flow_attr attr = { 0 };
  attr.group = 0;
  attr.priority = 1;
  attr.ingress = true;

  struct rte_flow_item_eth ethMask = { 0 };
  struct rte_flow_item_eth ethSpec = { 0 };
  if (sender == NULL) { // multicast
    memset(&ethMask.dst, 0xFF, ETHER_ADDR_LEN);
    memcpy(&ethSpec.dst, NDN_ETHER_MCAST, ETHER_ADDR_LEN);
  } else { // unicast
    memset(&ethMask.src, 0xFF, ETHER_ADDR_LEN);
    memcpy(&ethSpec.src, sender, ETHER_ADDR_LEN);
  }
  ethMask.type = 0xFFFF;
  ethSpec.type = rte_cpu_to_be_16(NDN_ETHERTYPE);

  struct rte_flow_item pattern[2] = { 0 };
  pattern[0].type = RTE_FLOW_ITEM_TYPE_ETH;
  pattern[0].mask = &ethMask;
  pattern[0].spec = &ethSpec;
  pattern[1].type = RTE_FLOW_ITEM_TYPE_END;

  struct rte_flow_action_queue queue = { .index = rxf->queue };

  struct rte_flow_action actions[2] = { 0 };
  actions[0].type = RTE_FLOW_ACTION_TYPE_QUEUE;
  actions[0].conf = &queue;
  actions[1].type = RTE_FLOW_ACTION_TYPE_END;

  rxf->flow = rte_flow_create(rxf->port, &attr, pattern, actions, error);
  return rxf->flow != NULL;
}

uint16_t
EthRxFlow_RxBurst(RxGroup* rxg, struct rte_mbuf** pkts, uint16_t nPkts)
{
  EthRxFlow* rxf = (EthRxFlow*)rxg;
  uint16_t nRx = rte_eth_rx_burst(rxf->port, rxf->queue, pkts, nPkts);
  uint64_t now = rte_get_tsc_cycles();
  for (uint16_t i = 0; i < nRx; ++i) {
    struct rte_mbuf* frame = pkts[i];
    frame->port = rxf->face;
    // TODO offload timestamping to hardware where available
    frame->timestamp = now;
    rte_pktmbuf_adj(frame, sizeof(struct ether_hdr));
  }
  return nRx;
}
