#include "gtpip-table.h"
#include "../core/logger.h"
#include "face.h"

N_LOG_INIT(GtpipTable);

bool
GtpipTable_ProcessDownlink(GtpipTable* table, struct rte_mbuf* m) {
  if (unlikely(m->data_len < RTE_ETHER_HDR_LEN + sizeof(struct rte_ipv4_hdr))) {
    return false;
  }
  const struct rte_ether_hdr* eth = rte_pktmbuf_mtod(m, const struct rte_ether_hdr*);
  if (eth->ether_type != rte_cpu_to_be_16(RTE_ETHER_TYPE_IPV4)) {
    return false;
  }
  const struct rte_ipv4_hdr* ip = RTE_PTR_ADD(eth, RTE_ETHER_HDR_LEN);
  rte_be32_t ueIP = ip->dst_addr;

  void* hdata = NULL;
  int res = rte_hash_lookup_data(table->ipv4, &ueIP, &hdata);
  if (res < 0) {
    return false;
  }

  FaceID id = (FaceID)(uintptr_t)hdata;
  EthFacePriv* priv = Face_GetPriv(Face_Get(id));
  EthTxHdr* hdr = &priv->txHdr;
  EthTxHdr_Prepend(hdr, m, EthTxHdrFlagsGtpip);

  return true;
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
GtpipTable_ProcessUplink(GtpipTable* table, struct rte_mbuf* m) {
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
  const struct rte_ipv4_hdr* iip = RTE_PTR_ADD(eth, hdrLen);
  rte_be32_t ueIP = iip->src_addr;

  void* hdata = NULL;
  int res = rte_hash_lookup_data(table->ipv4, &ueIP, &hdata);
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
