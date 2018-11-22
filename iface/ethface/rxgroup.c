#include "rxgroup.h"

static bool
EthRxGroup_Accept(EthRxGroup* rxg, struct rte_mbuf* frame, uint64_t now)
{
  assert(frame->data_len >= sizeof(struct ether_hdr));
  const struct ether_hdr* eth =
    rte_pktmbuf_mtod(frame, const struct ether_hdr*);

  // TODO offload ethertype filtering to hardware where available
  if (unlikely(eth->ether_type != rte_cpu_to_be_16(NDN_ETHERTYPE))) {
    rte_pktmbuf_free(frame);
    return false;
  }

  bool isMulticast = eth->d_addr.addr_bytes[0] & 0x01;
  if (isMulticast) {
    frame->port = rxg->multicast;
  } else {
    uint8_t srcLastOctet = eth->s_addr.addr_bytes[5];
    frame->port = rxg->unicast[srcLastOctet];
  }

  // TODO offload timestamping to hardware where available
  frame->timestamp = now;

  rte_pktmbuf_adj(frame, sizeof(struct ether_hdr));
  return true;
}

uint16_t
EthRxGroup_RxBurst(RxGroup* rxg0, struct rte_mbuf** pkts, uint16_t nPkts)
{
  EthRxGroup* rxg = (EthRxGroup*)rxg0;

  uint16_t nInput = rte_eth_rx_burst(rxg->port, rxg->queue, pkts, nPkts);

  uint64_t now = rte_get_tsc_cycles();
  uint16_t nRx = 0;
  for (uint16_t i = 0; i < nInput; ++i) {
    struct rte_mbuf* frame = pkts[i];
    if (likely(EthRxGroup_Accept(rxg, frame, now))) {
      pkts[nRx++] = frame;
    }
  }
  return nRx;
}
