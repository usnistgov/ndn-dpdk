#include "rxmatch.h"
#include "hdr-impl.h"

__attribute__((nonnull)) static inline bool
MatchAlways(const EthRxMatch* match, const struct rte_mbuf* m) {
  return true;
}

__attribute__((nonnull)) static inline bool
MatchVlan(const EthRxMatch* match, const struct rte_mbuf* m) {
  const struct rte_vlan_hdr* vlanM =
    rte_pktmbuf_mtod_offset(m, const struct rte_vlan_hdr*, RTE_ETHER_HDR_LEN);
  const struct rte_vlan_hdr* vlanT = RTE_PTR_ADD(match->buf, RTE_ETHER_HDR_LEN);
  return match->l2len != RTE_ETHER_HDR_LEN + RTE_VLAN_HLEN ||
         (vlanM->eth_proto == vlanT->eth_proto &&
          (vlanM->vlan_tci & rte_cpu_to_be_16(0x0FFF)) == vlanT->vlan_tci);
}

__attribute__((nonnull)) static inline bool
MatchEtherUnicast(const EthRxMatch* match, const struct rte_mbuf* m) {
  // exact match on Ethernet and VLAN headers
  return memcmp(rte_pktmbuf_mtod(m, const uint8_t*), match->buf, RTE_ETHER_HDR_LEN) == 0 &&
         MatchVlan(match, m);
}

__attribute__((nonnull)) static inline bool
MatchEtherMulticast(const EthRxMatch* match, const struct rte_mbuf* m) {
  // Ethernet destination must be multicast, exact match on ether_type and VLAN header
  const struct rte_ether_hdr* ethM = rte_pktmbuf_mtod(m, const struct rte_ether_hdr*);
  const struct rte_ether_hdr* ethT = (const struct rte_ether_hdr*)match->buf;
  return rte_is_multicast_ether_addr(&ethM->dst_addr) && ethM->ether_type == ethT->ether_type &&
         MatchVlan(match, m);
}

__attribute__((nonnull)) static inline bool
MatchUdp(const EthRxMatch* match, const struct rte_mbuf* m) {
  // UDP or GTP: exact match on IP addresses and UDP port numbers
  // VXLAN: exact match on IP addresses only
  return MatchEtherUnicast(match, m) &&
         memcmp(rte_pktmbuf_mtod_offset(m, const uint8_t*, match->l3matchOff),
                RTE_PTR_ADD(match->buf, match->l3matchOff), match->l3matchLen) == 0;
}

__attribute__((nonnull)) static inline bool
MatchVxlan(const EthRxMatch* match, const struct rte_mbuf* m) {
  // exact match on UDP destination port, VNI, and inner Ethernet header
  const struct rte_udp_hdr* udpM =
    rte_pktmbuf_mtod_offset(m, const struct rte_udp_hdr*, match->udpOff);
  const struct rte_vxlan_hdr* vxlanM = RTE_PTR_ADD(udpM, sizeof(*udpM));
  const struct rte_ether_hdr* innerEthM = RTE_PTR_ADD(vxlanM, sizeof(*vxlanM));
  const struct rte_udp_hdr* udpT = RTE_PTR_ADD(match->buf, match->udpOff);
  const struct rte_vxlan_hdr* vxlanT = RTE_PTR_ADD(udpT, sizeof(*udpT));
  const struct rte_ether_hdr* innerEthT = RTE_PTR_ADD(vxlanT, sizeof(*vxlanT));
  return MatchUdp(match, m) && udpM->dst_port == udpT->dst_port &&
         (vxlanM->vx_vni & ~rte_cpu_to_be_32(0xFF)) == vxlanT->vx_vni &&
         memcmp(innerEthM, innerEthT, RTE_ETHER_HDR_LEN) == 0;
}

__attribute__((nonnull)) static inline bool
MatchGtp(const EthRxMatch* match, const struct rte_mbuf* m) {
  // exact match on TEID and QFI; type=1 for uplink
  // inner IPv4+UDP headers are ignored
  const EthGtpHdr* gtpM =
    rte_pktmbuf_mtod_offset(m, const EthGtpHdr*, match->udpOff + sizeof(struct rte_udp_hdr));
  const EthGtpHdr* gtpT = RTE_PTR_ADD(match->buf, match->udpOff + sizeof(struct rte_udp_hdr));
  return MatchUdp(match, m) && gtpM->hdr.teid == gtpT->hdr.teid && gtpM->hdr.e == 1 &&
         gtpM->ext.next_ext == 0x85 && gtpM->psc.type == 1 && gtpM->psc.qfi == gtpT->psc.qfi;
}

const EthRxMatch_MatchFunc EthRxMatch_MatchJmp[] = {
  [EthRxMatchActAlways] = MatchAlways,
  [EthRxMatchActEtherUnicast] = MatchEtherUnicast,
  [EthRxMatchActEtherMulticast] = MatchEtherMulticast,
  [EthRxMatchActUdp] = MatchUdp,
  [EthRxMatchActVxlan] = MatchVxlan,
  [EthRxMatchActGtp] = MatchGtp,
};

void
EthRxMatch_Prepare(EthRxMatch* match, const EthLocator* loc) {
  EthLocatorClass c = EthLocator_Classify(loc);

  *match = (const EthRxMatch){.act = EthRxMatchActAlways};
  if (c.etherType == 0) { // memif or passthru
    return;
  }

#define BUF_TAIL (RTE_PTR_ADD(match->buf, match->len))

  match->l2len = PutEtherVlanHdr(BUF_TAIL, &loc->remote, &loc->local, loc->vlan, c.etherType);
  match->len += match->l2len;
  match->act = c.multicast ? EthRxMatchActEtherMulticast : EthRxMatchActEtherUnicast;
  if (!c.udp) {
    return;
  }

  match->len += (c.v4 ? PutIpv4Hdr : PutIpv6Hdr)(BUF_TAIL, loc->remoteIP, loc->localIP);
  uint8_t l3addrsLen = c.v4 ? sizeof(struct rte_ipv4_hdr) - offsetof(struct rte_ipv4_hdr, src_addr)
                            : sizeof(struct rte_ipv6_hdr) - offsetof(struct rte_ipv6_hdr, src_addr);
  match->udpOff = match->len;
  match->len += PutUdpHdr(BUF_TAIL, loc->remoteUDP, loc->localUDP);
  match->act = EthRxMatchActUdp;
  match->l3matchOff = match->udpOff - l3addrsLen;
  match->l3matchLen = l3addrsLen + offsetof(struct rte_udp_hdr, dgram_len);

  switch (c.tunnel) {
    case 'V': {
      match->l3matchLen = l3addrsLen;
      match->len += PutVxlanHdr(BUF_TAIL, loc->vxlan);
      match->len += PutEtherVlanHdr(BUF_TAIL, &loc->innerRemote, &loc->innerLocal, 0, EtherTypeNDN);
      match->act = EthRxMatchActVxlan;
      break;
    }
    case 'G': {
      match->len += PutGtpHdr(BUF_TAIL, true, loc->ulTEID, loc->ulQFI);
      match->len += PutIpv4Hdr(BUF_TAIL, loc->innerLocalIP, loc->innerRemoteIP);
      match->len += PutUdpHdr(BUF_TAIL, UDPPortNDN, UDPPortNDN);
      match->act = EthRxMatchActGtp;
      break;
    }
  }

#undef BUF_TAIL
  NDNDPDK_ASSERT(match->len <= sizeof(match->buf));
}
