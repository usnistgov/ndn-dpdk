#include "rxmatch.h"
#include "hdr-impl.h"

// Variable naming convention in this file:
// layerM: layer header in mbuf.
// layerT: layer header in match template.

static __rte_always_inline EthRxMatchResult
BoolResult(bool isHit) {
  return isHit ? EthRxMatchResultHit : 0;
}

__attribute__((nonnull)) static inline EthRxMatchResult
MatchAlways(const EthRxMatch* match, const struct rte_mbuf* m) {
  return EthRxMatchResultHit;
}

__attribute__((nonnull)) static __rte_always_inline EthRxMatchResult
MatchVlan(const EthRxMatch* match, const struct rte_mbuf* m) {
  const struct rte_vlan_hdr* vlanM =
    rte_pktmbuf_mtod_offset(m, const struct rte_vlan_hdr*, RTE_ETHER_HDR_LEN);
  const struct rte_vlan_hdr* vlanT = RTE_PTR_ADD(match->buf, RTE_ETHER_HDR_LEN);
  return BoolResult(match->l2len != RTE_ETHER_HDR_LEN + RTE_VLAN_HLEN ||
                    (vlanM->eth_proto == vlanT->eth_proto &&
                     (vlanM->vlan_tci & rte_cpu_to_be_16(0x0FFF)) == vlanT->vlan_tci));
}

__attribute__((nonnull)) static inline EthRxMatchResult
MatchEtherUnicast(const EthRxMatch* match, const struct rte_mbuf* m) {
  // exact match on Ethernet and VLAN headers
  return BoolResult(memcmp(rte_pktmbuf_mtod(m, const uint8_t*), match->buf, RTE_ETHER_HDR_LEN) ==
                      0 &&
                    MatchVlan(match, m));
}

__attribute__((nonnull)) static inline EthRxMatchResult
MatchEtherMulticast(const EthRxMatch* match, const struct rte_mbuf* m) {
  // Ethernet destination must be multicast, exact match on ether_type and VLAN header
  const struct rte_ether_hdr* ethM = rte_pktmbuf_mtod(m, const struct rte_ether_hdr*);
  const struct rte_ether_hdr* ethT = (const struct rte_ether_hdr*)match->buf;
  return BoolResult(rte_is_multicast_ether_addr(&ethM->dst_addr) &&
                    ethM->ether_type == ethT->ether_type && MatchVlan(match, m));
}

__attribute__((nonnull)) static inline EthRxMatchResult
MatchIpUdp(const EthRxMatch* match, const struct rte_mbuf* m) {
  // UDP: exact match on IP addresses and UDP port numbers
  // VXLAN or GTP: exact match on IP addresses only
  return BoolResult(MatchEtherUnicast(match, m) &&
                    memcmp(rte_pktmbuf_mtod_offset(m, const uint8_t*, match->l3matchOff),
                           RTE_PTR_ADD(match->buf, match->l3matchOff), match->l3matchLen) == 0);
}

__attribute__((nonnull)) static inline EthRxMatchResult
MatchVxlan(const EthRxMatch* match, const struct rte_mbuf* m) {
  // exact match on UDP destination port, VNI, and inner Ethernet header
  const struct rte_udp_hdr* udpM =
    rte_pktmbuf_mtod_offset(m, const struct rte_udp_hdr*, match->udpOff);
  const struct rte_vxlan_hdr* vxlanM = RTE_PTR_ADD(udpM, sizeof(*udpM));
  const struct rte_ether_hdr* iethM = RTE_PTR_ADD(vxlanM, sizeof(*vxlanM));
  const struct rte_udp_hdr* udpT = RTE_PTR_ADD(match->buf, match->udpOff);
  const struct rte_vxlan_hdr* vxlanT = RTE_PTR_ADD(udpT, sizeof(*udpT));
  const struct rte_ether_hdr* iethT = RTE_PTR_ADD(vxlanT, sizeof(*vxlanT));
  return BoolResult(MatchIpUdp(match, m) && udpM->dst_port == udpT->dst_port &&
                    memcmp(vxlanM->vni, vxlanT->vni, 3) == 0 &&
                    memcmp(iethM, iethT, RTE_ETHER_HDR_LEN) == 0);
}

__attribute__((nonnull)) static __rte_always_inline EthRxMatchResult
MatchGtpCommon(const EthRxMatch* match, const struct rte_mbuf* m, bool checkOuter) {
  EthRxMatchResult res = 0;

  // exact match on UDP destination port, TEID, and QFI; require psc.type=1 for uplink
  const struct rte_udp_hdr* udpM =
    rte_pktmbuf_mtod_offset(m, const struct rte_udp_hdr*, match->udpOff);
  const EthGtpHdr* gtpM = RTE_PTR_ADD(udpM, sizeof(*udpM));
  const struct rte_udp_hdr* udpT = RTE_PTR_ADD(match->buf, match->udpOff);
  const EthGtpHdr* gtpT = RTE_PTR_ADD(udpT, sizeof(*udpT));
  if (checkOuter &&
      !(MatchIpUdp(match, m) && udpM->dst_port == udpT->dst_port && EthGtpHdr_IsUplink(gtpM) &&
        gtpM->hdr.teid == gtpT->hdr.teid && gtpM->psc.qfi == gtpT->psc.qfi)) {
    return res;
  }
  res |= EthRxMatchResultGtp;

  // exact match on inner IPv4 addresses and UDP port numbers
  const struct rte_ipv4_hdr* iipM = RTE_PTR_ADD(gtpM, sizeof(*gtpM));
  const struct rte_ipv4_hdr* iipT = RTE_PTR_ADD(gtpT, sizeof(*gtpT));
  if (memcmp(&iipM->src_addr, &iipT->src_addr, 2 * sizeof(uint32_t) + 2 * sizeof(uint16_t)) == 0) {
    res |= EthRxMatchResultHit;
  }

  return res;
}

__attribute__((nonnull)) static inline EthRxMatchResult
MatchGtp(const EthRxMatch* match, const struct rte_mbuf* m) {
  return MatchGtpCommon(match, m, true);
}

EthRxMatchResult
EthRxMatch_MatchGtpInner(const EthRxMatch* match, const struct rte_mbuf* m) {
  NDNDPDK_ASSERT(match->act == EthRxMatchActGtp);
  NDNDPDK_ASSERT(m->data_len >= match->len);
  return MatchGtpCommon(match, m, false);
}

const EthRxMatch_MatchFunc EthRxMatch_MatchJmp[] = {
  [EthRxMatchActAlways] = MatchAlways,
  [EthRxMatchActEtherUnicast] = MatchEtherUnicast,
  [EthRxMatchActEtherMulticast] = MatchEtherMulticast,
  [EthRxMatchActUdp] = MatchIpUdp,
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

  match->l2len = PutEtherVlanHdr(BUF_TAIL, loc->remote, loc->local, loc->vlan, c.etherType);
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
  match->l3matchLen = l3addrsLen;

  switch (c.tunnel) {
    case 'V': {
      match->len += PutVxlanHdr(BUF_TAIL, loc->vxlan);
      match->len += PutEtherVlanHdr(BUF_TAIL, loc->innerRemote, loc->innerLocal, 0, EtherTypeNDN);
      match->act = EthRxMatchActVxlan;
      break;
    }
    case 'G': {
      match->len += PutGtpHdr(BUF_TAIL, true, loc->ulTEID, loc->ulQFI);
      match->len += PutIpv4Hdr(BUF_TAIL, loc->innerRemoteIP, loc->innerLocalIP);
      match->len += PutUdpHdr(BUF_TAIL, UDPPortNDN, UDPPortNDN);
      match->act = EthRxMatchActGtp;
      break;
    }
    default: {
      match->l3matchLen += 2 * sizeof(rte_be16_t); // MatchIpUdp checks UDP ports
      break;
    }
  }

#undef BUF_TAIL
  NDNDPDK_ASSERT(match->len <= sizeof(match->buf));
}
