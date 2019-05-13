#include "rxtable.h"

#include <rte_ethdev.h>
#include <rte_ether.h>

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
    frame->port = atomic_load_explicit(&rxt->multicast, memory_order_relaxed);
  } else {
    uint8_t srcLastOctet = eth->s_addr.addr_bytes[5];
    frame->port =
      atomic_load_explicit(&rxt->unicast[srcLastOctet], memory_order_relaxed);
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
