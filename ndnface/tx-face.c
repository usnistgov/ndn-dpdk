#include "tx-face.h"

static uint16_t
TxFace_TxCallback(uint16_t port, uint16_t queue, struct rte_mbuf** pkts,
                  uint16_t nPkts, void* face0)
{
  TxFace* face = (TxFace*)(face0);

  for (uint16_t i = 0; i < nPkts; ++i) {
    struct rte_mbuf* pkt = pkts[i];
    ++face->nPkts[Packet_GetNdnPktType(pkt)];
    face->nOctets += pkt->pkt_len;
  }

  return nPkts;
}

bool
TxFace_Init(TxFace* face)
{
  int res = rte_eth_dev_get_mtu(face->port, &face->mtu);
  if (res != 0) {
    return false;
  }

  rte_eth_macaddr_get(face->port, &face->ethhdr.s_addr);
  memset(&face->ethhdr.d_addr, 0xFF, sizeof(face->ethhdr.d_addr));
  face->ethhdr.ether_type = rte_cpu_to_be_16(NDN_ETHERTYPE);

  face->__txCallback =
    rte_eth_add_tx_callback(face->port, face->queue, &TxFace_TxCallback, face);
  if (face->__txCallback == NULL) {
    return false;
  }

  return true;
}

void
TxFace_Close(TxFace* face)
{
  rte_eth_remove_tx_callback(face->port, face->queue, face->__txCallback);
  face->__txCallback = NULL;
}

uint16_t
TxFace_TxBurst(TxFace* face, struct rte_mbuf** pkts, uint16_t nPkts)
{
  assert(face->mtu > 0);

  // TODO fragmentation

  for (uint16_t i = 0; i < nPkts; ++i) {
    struct rte_mbuf* pkt = pkts[i];
    // TODO do not prepend because pkt may be shared
    struct ether_hdr* eth =
      (struct ether_hdr*)rte_pktmbuf_prepend(pkt, sizeof(struct ether_hdr));
    assert(eth != NULL);
    memcpy(eth, &face->ethhdr, sizeof(*eth));
  }

  uint16_t n = 0;
  while (n == 0) {
    n = rte_eth_tx_burst(face->port, face->queue, pkts, nPkts);
    // TODO internal queuing
  }
  return n;
}