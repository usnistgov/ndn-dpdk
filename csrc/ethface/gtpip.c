#include "gtpip.h"
#include "../core/logger.h"
#include "face.h"

N_LOG_INIT(EthGtpip);

uint64_t
EthGtpip_ProcessDownlinkBulk(EthGtpip* g, struct rte_mbuf* pkts[], uint32_t count) {
  NDNDPDK_ASSERT(count <= RTE_HASH_LOOKUP_BULK_MAX);

  uint32_t nLookups = 0;
  uint64_t mask = 0;
  const void* keys[RTE_HASH_LOOKUP_BULK_MAX] = {0};
  for (uint32_t i = 0; i < count; ++i) {
    struct rte_mbuf* pkt = pkts[i];
    if (unlikely(pkt->data_len < RTE_ETHER_HDR_LEN + sizeof(struct rte_ipv4_hdr))) {
      continue;
    }
    const struct rte_ether_hdr* eth = rte_pktmbuf_mtod(pkt, const struct rte_ether_hdr*);
    if (eth->ether_type != rte_cpu_to_be_16(RTE_ETHER_TYPE_IPV4)) {
      continue;
    }
    const struct rte_ipv4_hdr* ip = RTE_PTR_ADD(eth, RTE_ETHER_HDR_LEN);
    rte_bit_set(&mask, i);
    keys[nLookups++] = &ip->dst_addr;
  }
  if (nLookups == 0) {
    return 0;
  }

  uint64_t hMask = 0;
  void* hData[RTE_HASH_LOOKUP_BULK_MAX];
  int hHits = rte_hash_lookup_bulk_data(g->ipv4, keys, nLookups, &hMask, hData);
  if (unlikely(hHits <= 0)) {
    return 0;
  }

  nLookups = 0;
  for (uint32_t i = 0; i < count; ++i) {
    if (!rte_bit_test(&mask, i)) {
      continue;
    }
    uint32_t hIndex = nLookups++;
    if (unlikely(!rte_bit_test(&hMask, hIndex))) {
      rte_bit_clear(&mask, i);
      continue;
    }
    FaceID id = (FaceID)(uintptr_t)(hData[hIndex]);
    EthFacePriv* priv = Face_GetPriv(Face_Get(id));
    EthTxHdr* hdr = &priv->txHdr;
    struct rte_mbuf* pkt = pkts[i];
    EthTxHdr_Prepend(hdr, pkt, EthTxHdrFlagsGtpip);
  }
  return mask;
}

// Uplink header lengths, from outer Ethernet to inner IPv4.
enum {
  UlHdrLenBase = RTE_ETHER_HDR_LEN + sizeof(struct rte_udp_hdr) + sizeof(EthGtpHdr) +
                 sizeof(struct rte_ipv4_hdr),
  UlHdrLenIpv4 = UlHdrLenBase + sizeof(struct rte_ipv4_hdr),
  UlHdrLenVlanIpv4 = UlHdrLenIpv4 + sizeof(struct rte_vlan_hdr),
  UlHdrLenIpv6 = UlHdrLenBase + sizeof(struct rte_ipv6_hdr),
  UlHdrLenVlanIpv6 = UlHdrLenIpv6 + sizeof(struct rte_vlan_hdr),
};

bool
EthGtpip_ProcessUplink(EthGtpip* g, struct rte_mbuf* m) {
  const struct rte_ether_hdr* eth = rte_pktmbuf_mtod(m, const struct rte_ether_hdr*);
  const struct rte_vlan_hdr* vlan = RTE_PTR_ADD(eth, RTE_ETHER_HDR_LEN);
  uint16_t hdrLen = 0;
  if (likely(m->data_len >= UlHdrLenIpv4) &&
      eth->ether_type == rte_cpu_to_be_16(RTE_ETHER_TYPE_IPV4)) {
    hdrLen = UlHdrLenIpv4;
  } else if (likely(m->data_len >= UlHdrLenVlanIpv4) &&
             eth->ether_type == rte_cpu_to_be_16(RTE_ETHER_TYPE_VLAN) &&
             vlan->eth_proto == rte_cpu_to_be_16(RTE_ETHER_TYPE_IPV4)) {
    hdrLen = UlHdrLenVlanIpv4;
  } else if (likely(m->data_len >= UlHdrLenIpv6) &&
             eth->ether_type == rte_cpu_to_be_16(RTE_ETHER_TYPE_IPV6)) {
    hdrLen = UlHdrLenIpv6;
  } else if (likely(m->data_len >= UlHdrLenVlanIpv6) &&
             eth->ether_type == rte_cpu_to_be_16(RTE_ETHER_TYPE_VLAN) &&
             vlan->eth_proto == rte_cpu_to_be_16(RTE_ETHER_TYPE_IPV6)) {
    hdrLen = UlHdrLenVlanIpv6;
  } else {
    return false;
  }
  hdrLen -= sizeof(struct rte_ipv4_hdr); // keep inner IPv4 header
  const struct rte_udp_hdr* udp = RTE_PTR_ADD(eth, hdrLen - sizeof(EthGtpHdr) - sizeof(*udp));
  if (unlikely(udp->src_port != rte_cpu_to_be_16(RTE_GTPU_UDP_PORT)) ||
      unlikely(udp->dst_port != rte_cpu_to_be_16(RTE_GTPU_UDP_PORT))) {
    return false;
  }
  const struct rte_ipv4_hdr* iip = RTE_PTR_ADD(eth, hdrLen);
  rte_be32_t ueIP = iip->src_addr;

  void* hdata = NULL;
  int res = rte_hash_lookup_data(g->ipv4, &ueIP, &hdata);
  if (res < 0) {
    return false;
  }

  FaceID id = (FaceID)(uintptr_t)hdata;
  EthFacePriv* priv = Face_GetPriv(Face_Get(id));
  EthRxMatch* match = &priv->rxMatch;

  if (unlikely(!EthRxMatch_MatchGtpip(match, m))) {
    return false;
  }

  struct rte_ether_hdr* eth1 =
    (struct rte_ether_hdr*)rte_pktmbuf_adj(m, hdrLen - RTE_ETHER_HDR_LEN);
  eth1->dst_addr = eth->dst_addr; // TAP netif has same MAC address as physical EthDev
  eth1->src_addr = eth->src_addr;
  eth1->ether_type = rte_cpu_to_be_16(RTE_ETHER_TYPE_IPV4);
  return true;
}
