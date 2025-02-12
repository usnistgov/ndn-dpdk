#include "locator.h"
#include "../ndni/an.h"

static const struct rte_ipv6_addr V4_IN_V6_PREFIX = RTE_IPV6_ADDR_PREFIX_V4MAPPED;
enum {
  IP_HOPLIMIT_VALUE = 64,
  VXLAN_SRCPORT_BASE = 0xC000,
  VXLAN_SRCPORT_MASK = 0x3FFF,
  V4_IN_V6_PREFIX_OCTETS = 12,
  V4_IN_V6_PREFIX_BITS = V4_IN_V6_PREFIX_OCTETS * CHAR_BIT,
};
static RTE_LCORE_VAR_HANDLE(uint16_t, txVxlanSrcPort);
RTE_LCORE_VAR_INIT(txVxlanSrcPort)

EthLocatorClass
EthLocator_Classify(const EthLocator* loc) {
  EthLocatorClass c = {0};
  if (rte_is_zero_ether_addr(&loc->local)) {
    if (rte_is_broadcast_ether_addr(&loc->remote)) {
      c.passthru = true;
    }
    return c;
  }
  c.multicast = rte_is_multicast_ether_addr(&loc->remote);
  c.udp = loc->remoteUDP != 0;
  // as of DPDK 24.11, rte_ipv6_addr_is_v4mapped has a bug:
  // it is passing depth=32 to rte_ipv6_addr_eq_prefix, should be depth=96
  c.v4 = rte_ipv6_addr_eq_prefix(&loc->remoteIP, &V4_IN_V6_PREFIX, V4_IN_V6_PREFIX_BITS);
  c.tunnel = 0;
  if (!rte_is_zero_ether_addr(&loc->innerRemote)) {
    c.tunnel = 'V';
  } else if (loc->isGtp) {
    c.tunnel = 'G';
  }
  c.etherType = !c.udp ? EtherTypeNDN : c.v4 ? RTE_ETHER_TYPE_IPV4 : RTE_ETHER_TYPE_IPV6;
  return c;
}

bool
EthLocator_CanCoexist(const EthLocator* a, const EthLocator* b) {
  EthLocatorClass ac = EthLocator_Classify(a);
  EthLocatorClass bc = EthLocator_Classify(b);
  if ((ac.etherType == 0 && !ac.passthru) || (bc.etherType == 0 && !bc.passthru)) {
    // only one memif face allowed
    return false;
  }
  if (ac.passthru || bc.passthru) {
    // only one passthru face allowed
    // passthru and non-passthru can coexist
    return ac.passthru != bc.passthru;
  }
  if (ac.multicast != bc.multicast || ac.udp != bc.udp || ac.v4 != bc.v4) {
    // Ethernet unicast and multicast can coexist
    // Ethernet, IPv4-UDP, and IPv6-UDP can coexist
    return true;
  }
  if (ac.multicast) {
    // only one Ethernet multicast face allowed
    return false;
  }
  if (a->vlan != b->vlan) {
    // different VLAN can coexist
    return true;
  }
  if (!ac.udp) {
    if (rte_is_same_ether_addr(&a->local, &b->local) &&
        rte_is_same_ether_addr(&a->remote, &b->remote)) {
      // Ethernet faces with same MAC addresses and VLAN conflict
      return false;
    }
    // Ethernet faces with different unicast MAC addresses can coexist
    return true;
  }
  if (!rte_ipv6_addr_eq(&a->localIP, &b->localIP) ||
      !rte_ipv6_addr_eq(&a->remoteIP, &b->remoteIP)) {
    // different IP addresses can coexist
    return true;
  }
  if (ac.tunnel == 0 && bc.tunnel == 0) {
    // UDP faces can coexist if either port number differs
    return a->localUDP != b->localUDP || a->remoteUDP != b->remoteUDP;
  }
  if (a->localUDP != b->localUDP && a->remoteUDP != b->remoteUDP) {
    // UDP face and VXLAN/GTP-U face -or- two VXLAN/GTP-U faces can coexist if both port numbers
    // differ
    return true;
  }
  if (ac.tunnel != bc.tunnel) {
    // UDP face and VXLAN face and GTP-U face with same port numbers conflict
    return false;
  }
  if (ac.tunnel == 'V') {
    // VXLAN faces can coexist if VNI or inner MAC address differ
    return a->vxlan != b->vxlan || !rte_is_same_ether_addr(&a->innerLocal, &b->innerLocal) ||
           !rte_is_same_ether_addr(&a->innerRemote, &b->innerRemote);
  }
  if (ac.tunnel == 'G') {
    // GTP-U faces can coexist if TEID differ
    return a->ulTEID != b->ulTEID && a->dlTEID != b->dlTEID;
  }
  NDNDPDK_ASSERT(false);
}

__attribute__((nonnull)) static uint8_t
PutEtherHdr(uint8_t* buffer, const struct rte_ether_addr* src, const struct rte_ether_addr* dst,
            uint16_t vid, uint16_t etherType) {
  struct rte_ether_hdr* ether = (struct rte_ether_hdr*)buffer;
  rte_ether_addr_copy(dst, &ether->dst_addr);
  rte_ether_addr_copy(src, &ether->src_addr);
  ether->ether_type = rte_cpu_to_be_16(vid == 0 ? etherType : RTE_ETHER_TYPE_VLAN);
  return RTE_ETHER_HDR_LEN;
}

__attribute__((nonnull)) static uint8_t
PutVlanHdr(uint8_t* buffer, uint16_t vid, uint16_t etherType) {
  struct rte_vlan_hdr* vlan = (struct rte_vlan_hdr*)buffer;
  vlan->vlan_tci = rte_cpu_to_be_16(vid);
  vlan->eth_proto = rte_cpu_to_be_16(etherType);
  return RTE_VLAN_HLEN;
}

__attribute__((nonnull)) static uint8_t
PutEtherVlanHdr(uint8_t* buffer, const struct rte_ether_addr* src, const struct rte_ether_addr* dst,
                uint16_t vid, uint16_t etherType) {
  uint8_t off = PutEtherHdr(buffer, src, dst, vid, etherType);
  if (vid != 0) {
    off += PutVlanHdr(RTE_PTR_ADD(buffer, off), vid, etherType);
  }
  return off;
}

__attribute__((nonnull)) static uint8_t
PutIpv4Hdr(uint8_t* buffer, const struct rte_ipv6_addr src, const struct rte_ipv6_addr dst) {
  struct rte_ipv4_hdr* ip = (struct rte_ipv4_hdr*)buffer;
  ip->version_ihl = RTE_IPV4_VHL_DEF;
  ip->fragment_offset = rte_cpu_to_be_16(RTE_IPV4_HDR_DF_FLAG);
  ip->time_to_live = IP_HOPLIMIT_VALUE;
  ip->next_proto_id = IPPROTO_UDP;
  rte_memcpy(&ip->src_addr, &src.a[V4_IN_V6_PREFIX_OCTETS], sizeof(ip->src_addr));
  rte_memcpy(&ip->dst_addr, &dst.a[V4_IN_V6_PREFIX_OCTETS], sizeof(ip->dst_addr));
  return sizeof(*ip);
}

__attribute__((nonnull)) static uint8_t
PutIpv6Hdr(uint8_t* buffer, const struct rte_ipv6_addr src, const struct rte_ipv6_addr dst) {
  struct rte_ipv6_hdr* ip = (struct rte_ipv6_hdr*)buffer;
  ip->vtc_flow = rte_cpu_to_be_32(6 << 28); // IP version 6
  ip->proto = IPPROTO_UDP;
  ip->hop_limits = IP_HOPLIMIT_VALUE;
  ip->src_addr = src;
  ip->dst_addr = dst;
  return sizeof(*ip);
}

__attribute__((nonnull)) static uint16_t
PutUdpHdr(uint8_t* buffer, uint16_t src, uint16_t dst) {
  struct rte_udp_hdr* udp = (struct rte_udp_hdr*)buffer;
  udp->src_port = rte_cpu_to_be_16(src);
  udp->dst_port = rte_cpu_to_be_16(dst);
  return sizeof(*udp);
}

__attribute__((nonnull)) static uint8_t
PutVxlanHdr(uint8_t* buffer, uint32_t vni) {
  struct rte_vxlan_hdr* vxlan = (struct rte_vxlan_hdr*)buffer;
  vxlan->vx_flags = rte_cpu_to_be_32(0x08000000);
  vxlan->vx_vni = rte_cpu_to_be_32(vni << 8);
  return sizeof(*vxlan);
}

__attribute__((nonnull)) static void
PutGtpHdrMinimal(struct rte_gtp_hdr* hdr, uint32_t teid) {
  hdr->ver = 1;
  hdr->pt = 1;
  hdr->e = 1;
  hdr->msg_type = 0xFF;
  hdr->teid = rte_cpu_to_be_32(teid);
}

__attribute__((nonnull)) static uint8_t
PutGtpHdr(uint8_t* buffer, bool ul, uint32_t teid, uint8_t qfi) {
  EthGtpHdr* gtp = (EthGtpHdr*)buffer;
  static_assert(sizeof(*gtp) == 16, "");
  PutGtpHdrMinimal(&gtp->hdr, teid);
  gtp->ext.next_ext = 0x85;
  gtp->psc.ext_hdr_len = 1;
  gtp->psc.type = (int)ul;
  gtp->psc.qfi = qfi;
  return sizeof(*gtp);
}

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
  const EthGtpHdr* gtpM =
    rte_pktmbuf_mtod_offset(m, const EthGtpHdr*, match->udpOff + sizeof(struct rte_udp_hdr));
  const EthGtpHdr* gtpT = RTE_PTR_ADD(match->buf, match->udpOff + sizeof(struct rte_udp_hdr));
  return MatchUdp(match, m) && gtpM->hdr.teid == gtpT->hdr.teid && gtpM->hdr.e == 1 &&
         gtpM->ext.next_ext == 0x85 && gtpM->psc.type == 1 && gtpM->psc.qfi == gtpT->psc.qfi;
}

void
EthRxMatch_Prepare(EthRxMatch* match, const EthLocator* loc) {
  EthLocatorClass c = EthLocator_Classify(loc);

  *match = (const EthRxMatch){.f = MatchAlways};
  if (c.etherType == 0) { // memif or passthru
    return;
  }

#define BUF_TAIL (RTE_PTR_ADD(match->buf, match->len))

  match->l2len = PutEtherVlanHdr(BUF_TAIL, &loc->remote, &loc->local, loc->vlan, c.etherType);
  match->len += match->l2len;
  match->f = c.multicast ? MatchEtherMulticast : MatchEtherUnicast;
  if (!c.udp) {
    return;
  }

  match->len += (c.v4 ? PutIpv4Hdr : PutIpv6Hdr)(BUF_TAIL, loc->remoteIP, loc->localIP);
  uint8_t l3addrsLen = c.v4 ? sizeof(struct rte_ipv4_hdr) - offsetof(struct rte_ipv4_hdr, src_addr)
                            : sizeof(struct rte_ipv6_hdr) - offsetof(struct rte_ipv6_hdr, src_addr);
  match->udpOff = match->len;
  match->len += PutUdpHdr(BUF_TAIL, loc->remoteUDP, loc->localUDP);
  match->f = MatchUdp;
  match->l3matchOff = match->udpOff - l3addrsLen;
  match->l3matchLen = l3addrsLen + offsetof(struct rte_udp_hdr, dgram_len);

  switch (c.tunnel) {
    case 'V': {
      match->l3matchLen = l3addrsLen;
      match->len += PutVxlanHdr(BUF_TAIL, loc->vxlan);
      match->len += PutEtherVlanHdr(BUF_TAIL, &loc->innerRemote, &loc->innerLocal, 0, EtherTypeNDN);
      match->f = MatchVxlan;
      break;
    }
    case 'G': {
      match->len += PutGtpHdr(BUF_TAIL, true, loc->ulTEID, loc->ulQFI);
      match->len += PutIpv4Hdr(BUF_TAIL, loc->innerLocalIP, loc->innerRemoteIP);
      match->len += PutUdpHdr(BUF_TAIL, UDPPortNDN, UDPPortNDN);
      match->f = MatchGtp;
      break;
    }
  }

#undef BUF_TAIL
  NDNDPDK_ASSERT(match->len <= sizeof(match->buf));
}

void
EthXdpLocator_Prepare(EthXdpLocator* xl, const EthLocator* loc) {
  EthLocatorClass c = EthLocator_Classify(loc);

  *xl = (const EthXdpLocator){0};
  if (c.etherType == 0) {
    return;
  }

  if (c.multicast) {
    rte_memcpy(xl->ether, &loc->remote, RTE_ETHER_ADDR_LEN);
  } else {
    rte_memcpy(xl->ether, &loc->local, RTE_ETHER_ADDR_LEN);
    rte_memcpy(RTE_PTR_ADD(xl->ether, RTE_ETHER_ADDR_LEN), &loc->remote, RTE_ETHER_ADDR_LEN);
  }
  if (loc->vlan != 0) {
    xl->vlan = rte_cpu_to_be_16(loc->vlan);
  }
  if (!c.udp) {
    return;
  }

  if (c.v4) {
    rte_memcpy(xl->ip, RTE_PTR_ADD(loc->remoteIP.a, V4_IN_V6_PREFIX_OCTETS),
               RTE_SIZEOF_FIELD(struct rte_ipv4_hdr, src_addr));
    rte_memcpy(RTE_PTR_ADD(xl->ip, RTE_SIZEOF_FIELD(struct rte_ipv4_hdr, src_addr)),
               RTE_PTR_ADD(loc->localIP.a, V4_IN_V6_PREFIX_OCTETS),
               RTE_SIZEOF_FIELD(struct rte_ipv4_hdr, dst_addr));
  } else {
    rte_memcpy(xl->ip, loc->remoteIP.a, RTE_IPV6_ADDR_SIZE);
    rte_memcpy(RTE_PTR_ADD(xl->ip, RTE_IPV6_ADDR_SIZE), loc->localIP.a, RTE_IPV6_ADDR_SIZE);
  }
  xl->udpSrc = rte_cpu_to_be_16(loc->remoteUDP);
  xl->udpDst = rte_cpu_to_be_16(loc->localUDP);
  switch (c.tunnel) {
    case 'V': {
      xl->udpSrc = 0;
      xl->vxlan = rte_cpu_to_be_32(loc->vxlan << 8);
      rte_memcpy(xl->inner, &loc->innerLocal, RTE_ETHER_ADDR_LEN);
      rte_memcpy(RTE_PTR_ADD(xl->inner, RTE_ETHER_ADDR_LEN), &loc->innerRemote, RTE_ETHER_ADDR_LEN);
      break;
    }
    case 'G': {
      xl->teid = rte_cpu_to_be_32(loc->ulTEID);
      xl->qfi = loc->ulQFI;
      break;
    }
  }
}

static void
EthFlowPattern_Set(EthFlowPattern* flow, size_t i, enum rte_flow_item_type typ, uint8_t* spec,
                   uint8_t* mask, size_t size) {
  for (size_t j = 0; j < size; ++j) {
    spec[j] &= mask[j];
  }
  flow->pattern[i].type = typ;
  flow->pattern[i].spec = spec;
  flow->pattern[i].mask = mask;
}

void
EthFlowPattern_Prepare(EthFlowPattern* flow, uint32_t* priority, const EthLocator* loc,
                       bool prefersFlowItemGTP) {
  EthLocatorClass c = EthLocator_Classify(loc);

  *flow = (const EthFlowPattern){0};
  flow->pattern[0].type = RTE_FLOW_ITEM_TYPE_END;
  *priority = 0;
  size_t i = 0;
#define MASK(field) memset(&(field), 0xFF, sizeof(field))
#define APPEND(typ, field)                                                                         \
  do {                                                                                             \
    EthFlowPattern_Set(flow, i, RTE_FLOW_ITEM_TYPE_##typ, (uint8_t*)&flow->field##Spec,            \
                       (uint8_t*)&flow->field##Mask, sizeof(flow->field##Mask));                   \
    ++i;                                                                                           \
    NDNDPDK_ASSERT(i < RTE_DIM(flow->pattern));                                                    \
    flow->pattern[i].type = RTE_FLOW_ITEM_TYPE_END;                                                \
  } while (false)

  if (c.passthru) {
    *priority = 1;
    return;
  }

  MASK(flow->ethMask.hdr.dst_addr);
  MASK(flow->ethMask.hdr.ether_type);
  PutEtherHdr((uint8_t*)(&flow->ethSpec.hdr), &loc->remote, &loc->local, loc->vlan, c.etherType);
  if (c.multicast) {
    rte_ether_addr_copy(&loc->remote, &flow->ethSpec.hdr.dst_addr);
  } else {
    MASK(flow->ethMask.hdr.src_addr);
  }
  APPEND(ETH, eth);

  if (loc->vlan != 0) {
    flow->vlanMask.hdr.vlan_tci = rte_cpu_to_be_16(0x0FFF); // don't mask PCP & DEI bits
    MASK(flow->vlanMask.hdr.eth_proto);
    PutVlanHdr((uint8_t*)(&flow->vlanSpec.hdr), loc->vlan, c.etherType);
    APPEND(VLAN, vlan);
  }

  if (!c.udp) {
    MASK(flow->vlanMask.hdr.eth_proto);
    return;
  }
  // several drivers do not support ETH+IP combination, so clear ETH spec
  flow->pattern[0].spec = NULL;
  flow->pattern[0].mask = NULL;

  if (c.v4) {
    MASK(flow->ip4Mask.hdr.src_addr);
    MASK(flow->ip4Mask.hdr.dst_addr);
    PutIpv4Hdr((uint8_t*)(&flow->ip4Spec.hdr), loc->remoteIP, loc->localIP);
    APPEND(IPV4, ip4);
  } else {
    MASK(flow->ip6Mask.hdr.src_addr);
    MASK(flow->ip6Mask.hdr.dst_addr);
    PutIpv6Hdr((uint8_t*)(&flow->ip6Spec.hdr), loc->remoteIP, loc->localIP);
    APPEND(IPV6, ip6);
  }

  if (c.tunnel != 'V') { // VXLAN packet can have any UDP source port
    MASK(flow->udpMask.hdr.dst_port);
  }
  MASK(flow->udpMask.hdr.src_port);
  PutUdpHdr((uint8_t*)(&flow->udpSpec.hdr), loc->remoteUDP, loc->localUDP);
  APPEND(UDP, udp);

  switch (c.tunnel) {
    case 'V': {
      flow->vxlanMask.hdr.vx_vni = ~rte_cpu_to_be_32(0xFF); // don't mask reserved byte
      PutVxlanHdr((uint8_t*)(&flow->vxlanSpec.hdr), loc->vxlan);
      APPEND(VXLAN, vxlan);

      MASK(flow->innerEthMask.hdr.dst_addr);
      MASK(flow->innerEthMask.hdr.src_addr);
      MASK(flow->innerEthMask.hdr.ether_type);
      PutEtherHdr((uint8_t*)(&flow->innerEthSpec.hdr), &loc->innerRemote, &loc->innerLocal, 0,
                  EtherTypeNDN);
      APPEND(ETH, innerEth);
      break;
    }
    case 'G': {
      MASK(flow->gtpMask.hdr.teid);
      PutGtpHdrMinimal(&flow->gtpSpec.hdr, loc->ulTEID);
      if (prefersFlowItemGTP) {
        APPEND(GTP, gtp);
      } else {
        APPEND(GTPU, gtp);
      }
      break;
    }
  }

#undef MASK
#undef APPEND
}

__attribute__((nonnull)) static void
TxNoHdr(const EthTxHdr* hdr, struct rte_mbuf* m, bool newBurst) {}

__attribute__((nonnull)) static __rte_always_inline void
TxPrepend(const EthTxHdr* hdr, struct rte_mbuf* m) {
  char* room = rte_pktmbuf_prepend(m, hdr->len);
  NDNDPDK_ASSERT(room != NULL); // enough headroom is required
  rte_memcpy(room, hdr->buf, hdr->len);
}

__attribute__((nonnull)) static void
TxEther(const EthTxHdr* hdr, struct rte_mbuf* m, bool newBurst) {
  TxPrepend(hdr, m);
}

__attribute__((nonnull)) static __rte_always_inline void
TxUdpCommon(const EthTxHdr* hdr, struct rte_udp_hdr* udp, uint16_t udpLen, bool newBurst) {
  udp->dgram_len = rte_cpu_to_be_16(udpLen);
  switch (hdr->tunnel) {
    case 'V': {
      static_assert((VXLAN_SRCPORT_BASE & VXLAN_SRCPORT_MASK) == 0, "");
      uint16_t srcPort = (*LCORE_VAR_SAFE(txVxlanSrcPort) += (uint16_t)newBurst);
      udp->src_port = rte_cpu_to_be_16((srcPort & VXLAN_SRCPORT_MASK) | VXLAN_SRCPORT_BASE);
      break;
    }
    case 'G': {
      EthGtpHdr* gtp = RTE_PTR_ADD(udp, sizeof(*udp));
      struct rte_ipv4_hdr* ip = RTE_PTR_ADD(gtp, sizeof(*gtp));
      struct rte_udp_hdr* iudp = RTE_PTR_ADD(ip, sizeof(*ip));
      uint16_t gtpLen = udpLen - sizeof(*udp);
      gtp->hdr.plen = rte_cpu_to_be_16(gtpLen - sizeof(gtp->hdr));
      ip->total_length = rte_cpu_to_be_16(gtpLen - sizeof(*gtp));
      iudp->dgram_len = rte_cpu_to_be_16(gtpLen - sizeof(*gtp) - sizeof(*ip));
      ip->hdr_checksum = rte_ipv4_cksum(ip);
      break;
    }
  }
}

__attribute__((nonnull)) static __rte_always_inline struct rte_ipv4_hdr*
TxUdp4(const EthTxHdr* hdr, struct rte_mbuf* m, bool newBurst) {
  TxPrepend(hdr, m);
  struct rte_ipv4_hdr* ip = rte_pktmbuf_mtod_offset(m, struct rte_ipv4_hdr*, hdr->l2len);
  struct rte_udp_hdr* udp = RTE_PTR_ADD(ip, sizeof(*ip));
  uint16_t ipLen = m->pkt_len - hdr->l2len;
  ip->total_length = rte_cpu_to_be_16(ipLen);
  TxUdpCommon(hdr, udp, ipLen - sizeof(*ip), newBurst);
  return ip;
}

__attribute__((nonnull)) static void
TxUdp4Checksum(const EthTxHdr* hdr, struct rte_mbuf* m, bool newBurst) {
  struct rte_ipv4_hdr* ip = TxUdp4(hdr, m, newBurst);
  ip->hdr_checksum = rte_ipv4_cksum(ip);
}

__attribute__((nonnull)) static void
TxUdp4Offload(const EthTxHdr* hdr, struct rte_mbuf* m, bool newBurst) {
  struct rte_ipv4_hdr* ip = TxUdp4(hdr, m, newBurst);
  m->l2_len = hdr->l2len;
  m->l3_len = sizeof(*ip);
  m->ol_flags |= RTE_MBUF_F_TX_IPV4 | RTE_MBUF_F_TX_IP_CKSUM;
}

__attribute__((nonnull)) static __rte_always_inline struct rte_ipv6_hdr*
TxUdp6(const EthTxHdr* hdr, struct rte_mbuf* m, bool newBurst) {
  TxPrepend(hdr, m);
  struct rte_ipv6_hdr* ip = rte_pktmbuf_mtod_offset(m, struct rte_ipv6_hdr*, hdr->l2len);
  struct rte_udp_hdr* udp = RTE_PTR_ADD(ip, sizeof(*ip));
  TxUdpCommon(hdr, udp, m->pkt_len - hdr->l2len - sizeof(*ip), newBurst);
  ip->payload_len = udp->dgram_len;
  return ip;
}

__attribute__((nonnull)) static void
TxUdp6Checksum(const EthTxHdr* hdr, struct rte_mbuf* m, bool newBurst) {
  struct rte_ipv6_hdr* ip = TxUdp6(hdr, m, newBurst);
  struct rte_udp_hdr* udp = RTE_PTR_ADD(ip, sizeof(*ip));
  udp->dgram_cksum = rte_ipv6_udptcp_cksum_mbuf(m, ip, hdr->l2len + sizeof(*ip));
}

__attribute__((nonnull)) static void
TxUdp6Offload(const EthTxHdr* hdr, struct rte_mbuf* m, bool newBurst) {
  struct rte_ipv6_hdr* ip = TxUdp6(hdr, m, newBurst);
  struct rte_udp_hdr* udp = RTE_PTR_ADD(ip, sizeof(*ip));
  m->l2_len = hdr->l2len;
  m->l3_len = sizeof(*ip);
  m->ol_flags |= RTE_MBUF_F_TX_IPV6 | RTE_MBUF_F_TX_UDP_CKSUM;
  udp->dgram_cksum = rte_ipv6_phdr_cksum(ip, m->ol_flags);
}

void
EthTxHdr_Prepare(EthTxHdr* hdr, const EthLocator* loc, bool hasChecksumOffloads) {
  EthLocatorClass c = EthLocator_Classify(loc);

  *hdr = (const EthTxHdr){.f = TxEther};
  if (c.etherType == 0) {
    hdr->f = TxNoHdr;
    return;
  }

#define BUF_TAIL (RTE_PTR_ADD(hdr->buf, hdr->len))

  hdr->l2len = PutEtherVlanHdr(BUF_TAIL, &loc->local, &loc->remote, loc->vlan, c.etherType);
  hdr->len += hdr->l2len;

  if (!c.udp) {
    return;
  }
  hdr->f = c.v4 ? (hasChecksumOffloads ? TxUdp4Offload : TxUdp4Checksum)
                : (hasChecksumOffloads ? TxUdp6Offload : TxUdp6Checksum);
  hdr->len += (c.v4 ? PutIpv4Hdr : PutIpv6Hdr)(BUF_TAIL, loc->localIP, loc->remoteIP);
  hdr->len += PutUdpHdr(BUF_TAIL, loc->localUDP, loc->remoteUDP);

  hdr->tunnel = c.tunnel;
  switch (c.tunnel) {
    case 'V': {
      hdr->len += PutVxlanHdr(BUF_TAIL, loc->vxlan);
      hdr->len += PutEtherVlanHdr(BUF_TAIL, &loc->innerLocal, &loc->innerRemote, 0, EtherTypeNDN);
      break;
    }
    case 'G': {
      hdr->len += PutGtpHdr(BUF_TAIL, false, loc->dlTEID, loc->dlQFI);
      hdr->len += PutIpv4Hdr(BUF_TAIL, loc->innerLocalIP, loc->innerRemoteIP);
      hdr->len += PutUdpHdr(BUF_TAIL, UDPPortNDN, UDPPortNDN);
      break;
    }
  }

#undef BUF_TAIL
  NDNDPDK_ASSERT(hdr->len <= sizeof(hdr->buf));
}
