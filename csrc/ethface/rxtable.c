#include "rxtable.h"
#include "eth-face.h"

static bool
EthRxTable_Accept(EthRxTable* rxt, struct rte_mbuf* frame, uint64_t now)
{
  assert(frame->data_len >= sizeof(EthFaceEtherHdr));
  const EthFaceEtherHdr* hdr = rte_pktmbuf_mtod(frame, const EthFaceEtherHdr*);

  if (rte_is_multicast_ether_addr(&hdr->eth.d_addr)) {
    frame->port = atomic_load_explicit(&rxt->multicast, memory_order_relaxed);
  } else {
    uint8_t srcLastOctet = hdr->eth.s_addr.addr_bytes[5];
    frame->port =
      atomic_load_explicit(&rxt->unicast[srcLastOctet], memory_order_relaxed);
  }

  if (likely(hdr->eth.ether_type == rte_cpu_to_be_16(NDN_ETHERTYPE))) {
    rte_pktmbuf_adj(frame, offsetof(EthFaceEtherHdr, vlan0));
  } else if (likely(hdr->vlan0.eth_proto == rte_cpu_to_be_16(NDN_ETHERTYPE))) {
    rte_pktmbuf_adj(frame, offsetof(EthFaceEtherHdr, vlan1));
  } else if (likely(hdr->vlan1.eth_proto == rte_cpu_to_be_16(NDN_ETHERTYPE))) {
    rte_pktmbuf_adj(frame, sizeof(EthFaceEtherHdr));
  } else {
    rte_pktmbuf_free(frame);
    return false;
  }

  frame->timestamp = now;
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
