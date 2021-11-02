#include "locator.h"
#include "../ndni/an.h"

#define ETHER_VLAN_HLEN (RTE_ETHER_HDR_LEN + sizeof(struct rte_vlan_hdr))
#define ETHER_ETHERTYPE_OFFSET (offsetof(struct rte_ether_hdr, ether_type))
#define VXLAN_ETHER_HLEN (sizeof(struct rte_vxlan_hdr) + RTE_ETHER_HDR_LEN)
#define IP_HOPLIMIT_VALUE 64
#define VXLAN_SRCPORT_BASE 0xC000
#define VXLAN_SRCPORT_MASK 0x3FFF
static const uint8_t V4_IN_V6_PREFIX[] = { 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
                                           0x00, 0x00, 0x00, 0x00, 0xFF, 0xFF };
static RTE_DEFINE_PER_LCORE(uint16_t, txVxlanSrcPort);

typedef struct ClassifyResult
{
  uint16_t etherType; ///< outer EtherType, 0 for memif
  bool multicast;     ///< is outer Ethernet multicast?
  bool udp;           ///< is UDP tunnel?
  bool v4;            ///< is IPv4?
  bool vxlan;         ///< is VXLAN?
} ClassifyResult;

__attribute__((nonnull)) static ClassifyResult
EthLocator_Classify(const EthLocator* loc)
{
  ClassifyResult c = { 0 };
  if (rte_is_zero_ether_addr(&loc->local)) {
    return c;
  }
  c.multicast = rte_is_multicast_ether_addr(&loc->remote);
  c.udp = loc->remoteUDP != 0;
  c.v4 = memcmp(loc->remoteIP, V4_IN_V6_PREFIX, sizeof(V4_IN_V6_PREFIX)) == 0;
  c.vxlan = !rte_is_zero_ether_addr(&loc->innerRemote);
  c.etherType = !c.udp ? EtherTypeNDN : c.v4 ? RTE_ETHER_TYPE_IPV4 : RTE_ETHER_TYPE_IPV6;
  return c;
}

bool
EthLocator_CanCoexist(const EthLocator* a, const EthLocator* b)
{
  ClassifyResult ac = EthLocator_Classify(a);
  ClassifyResult bc = EthLocator_Classify(b);
  if (ac.etherType == 0 || bc.etherType == 0) {
    return false;
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
  if (memcmp(a->localIP, b->localIP, sizeof(a->localIP)) != 0 ||
      memcmp(a->remoteIP, b->remoteIP, sizeof(a->remoteIP)) != 0) {
    // different IP addresses can coexist
    return true;
  }
  if (!ac.vxlan && !bc.vxlan) {
    // UDP faces can coexist if either port number differs
    return a->localUDP != b->localUDP || a->remoteUDP != b->remoteUDP;
  }
  if (a->localUDP != b->localUDP && a->remoteUDP != b->remoteUDP) {
    // UDP face an VXLAN face -or- two UDP faces can coexist if both port numbers differ
    return true;
  }
  if (ac.vxlan != bc.vxlan) {
    // UDP face and VXLAN face with same port numbers conflict
    return false;
  }
  // VXLAN faces can coexist if VNI or inner MAC address differ
  return a->vxlan != b->vxlan || !rte_is_same_ether_addr(&a->innerLocal, &b->innerLocal) ||
         !rte_is_same_ether_addr(&a->innerRemote, &b->innerRemote);
}

__attribute__((nonnull)) static uint8_t
PutEtherVlanHdr(uint8_t* buffer, const struct rte_ether_addr* src, const struct rte_ether_addr* dst,
                uint16_t vid, uint16_t etherType)
{
  struct rte_ether_hdr* ether = (struct rte_ether_hdr*)buffer;
  rte_ether_addr_copy(dst, &ether->d_addr);
  rte_ether_addr_copy(src, &ether->s_addr);
  ether->ether_type = rte_cpu_to_be_16(etherType);
  if (vid == 0) {
    return RTE_ETHER_HDR_LEN;
  }

  struct rte_vlan_hdr* vlan = (struct rte_vlan_hdr*)RTE_PTR_ADD(buffer, RTE_ETHER_HDR_LEN);
  vlan->vlan_tci = rte_cpu_to_be_16(vid & 0x0FFF);
  vlan->eth_proto = ether->ether_type;
  ether->ether_type = rte_cpu_to_be_16(RTE_ETHER_TYPE_VLAN);
  return ETHER_VLAN_HLEN;
}

__attribute__((nonnull)) static uint8_t
PutIpv4Hdr(uint8_t* buffer, const uint8_t* src, const uint8_t* dst)
{
  struct rte_ipv4_hdr* ip = (struct rte_ipv4_hdr*)buffer;
  ip->version_ihl = 0x45;                         // IPv4, header length 5 words
  ip->fragment_offset = rte_cpu_to_be_16(0x4000); // Don't Fragment
  ip->time_to_live = IP_HOPLIMIT_VALUE;
  ip->next_proto_id = IPPROTO_UDP;
  rte_memcpy(&ip->src_addr, RTE_PTR_ADD(src, sizeof(V4_IN_V6_PREFIX)), sizeof(ip->src_addr));
  rte_memcpy(&ip->dst_addr, RTE_PTR_ADD(dst, sizeof(V4_IN_V6_PREFIX)), sizeof(ip->dst_addr));
  return sizeof(*ip);
}

__attribute__((nonnull)) static uint8_t
PutIpv6Hdr(uint8_t* buffer, const uint8_t* src, const uint8_t* dst)
{
  struct rte_ipv6_hdr* ip = (struct rte_ipv6_hdr*)buffer;
  ip->vtc_flow = rte_cpu_to_be_32(6 << 28); // IPv6
  ip->proto = IPPROTO_UDP;
  ip->hop_limits = IP_HOPLIMIT_VALUE;
  rte_memcpy(ip->src_addr, src, sizeof(ip->src_addr));
  rte_memcpy(ip->dst_addr, dst, sizeof(ip->dst_addr));
  return sizeof(*ip);
}

__attribute__((nonnull)) static uint16_t
PutUdpHdr(uint8_t* buffer, uint16_t src, uint16_t dst)
{
  struct rte_udp_hdr* udp = (struct rte_udp_hdr*)buffer;
  udp->src_port = rte_cpu_to_be_16(src);
  udp->dst_port = rte_cpu_to_be_16(dst);
  return sizeof(*udp);
}

__attribute__((nonnull)) static uint8_t
PutVxlanHdr(uint8_t* buffer, uint32_t vni)
{
  struct rte_vxlan_hdr* vxlan = (struct rte_vxlan_hdr*)buffer;
  vxlan->vx_flags = rte_cpu_to_be_32(0x08000000);
  vxlan->vx_vni = rte_cpu_to_be_32(vni << 8);
  return sizeof(*vxlan);
}

__attribute__((nonnull)) static bool
MatchAlways(const EthRxMatch* match, const struct rte_mbuf* m)
{
  return true;
}

__attribute__((nonnull)) static __rte_always_inline bool
MatchVlan(const EthRxMatch* match, const struct rte_mbuf* m)
{
  const struct rte_vlan_hdr* vlan =
    rte_pktmbuf_mtod_offset(m, const struct rte_vlan_hdr*, RTE_ETHER_HDR_LEN);
  const struct rte_vlan_hdr* vlant = RTE_PTR_ADD(match->buf, RTE_ETHER_HDR_LEN);
  return match->l2len != ETHER_VLAN_HLEN ||
         (vlan->eth_proto == vlant->eth_proto &&
          (vlan->vlan_tci & rte_cpu_to_be_16(0x0FFF)) == vlant->vlan_tci);
}

__attribute__((nonnull)) static bool
MatchEtherUnicast(const EthRxMatch* match, const struct rte_mbuf* m)
{
  // exact match on Ethernet and VLAN headers
  return memcmp(rte_pktmbuf_mtod(m, const uint8_t*), match->buf, RTE_ETHER_HDR_LEN) == 0 &&
         MatchVlan(match, m);
}

__attribute__((nonnull)) static bool
MatchEtherMulticast(const EthRxMatch* match, const struct rte_mbuf* m)
{
  // Ethernet destination must be multicast, exact match on ether_type and VLAN header
  const struct rte_ether_hdr* ether = rte_pktmbuf_mtod(m, const struct rte_ether_hdr*);
  return rte_is_multicast_ether_addr(&ether->d_addr) &&
         memcmp(RTE_PTR_ADD(ether, ETHER_ETHERTYPE_OFFSET),
                RTE_PTR_ADD(match->buf, ETHER_ETHERTYPE_OFFSET),
                RTE_ETHER_HDR_LEN - ETHER_ETHERTYPE_OFFSET) == 0 &&
         MatchVlan(match, m);
}

__attribute__((nonnull)) static bool
MatchUdp(const EthRxMatch* match, const struct rte_mbuf* m)
{
  // UDP: exact match on IP addresses and UDP port numbers
  // VXLAN: exact match on IP addresses only
  return MatchEtherUnicast(match, m) &&
         memcmp(rte_pktmbuf_mtod_offset(m, const uint8_t*, match->l3matchOff),
                RTE_PTR_ADD(match->buf, match->l3matchOff), match->l3matchLen) == 0;
}

__attribute__((nonnull)) static bool
MatchVxlan(const EthRxMatch* match, const struct rte_mbuf* m)
{
  // exact match on UDP destination port, VNI, and inner Ethernet header
  const struct rte_udp_hdr* udp =
    rte_pktmbuf_mtod_offset(m, const struct rte_udp_hdr*, match->udpOff);
  const struct rte_vxlan_hdr* vxlan = RTE_PTR_ADD(udp, sizeof(*udp));
  const struct rte_ether_hdr* ieth = RTE_PTR_ADD(vxlan, sizeof(*vxlan));
  const struct rte_udp_hdr* udpt = RTE_PTR_ADD(match->buf, match->udpOff);
  const struct rte_vxlan_hdr* vxlant = RTE_PTR_ADD(udpt, sizeof(*udpt));
  const struct rte_ether_hdr* ietht = RTE_PTR_ADD(vxlant, sizeof(*vxlant));
  return MatchUdp(match, m) && udp->dst_port == udpt->dst_port &&
         (vxlan->vx_vni & ~rte_cpu_to_be_32(0xFF)) == vxlant->vx_vni &&
         memcmp(ieth, ietht, RTE_ETHER_HDR_LEN) == 0;
}

void
EthRxMatch_Prepare(EthRxMatch* match, const EthLocator* loc)
{
  ClassifyResult classify = EthLocator_Classify(loc);

  *match = (const EthRxMatch){ 0 };
  if (classify.etherType == 0) {
    match->f = MatchAlways;
    return;
  }

#define BUF_TAIL (RTE_PTR_ADD(match->buf, match->len))

  match->l2len =
    PutEtherVlanHdr(BUF_TAIL, &loc->remote, &loc->local, loc->vlan, classify.etherType);
  match->len += match->l2len;
  match->f = classify.multicast ? MatchEtherMulticast : MatchEtherUnicast;
  if (!classify.udp) {
    return;
  }

  match->len += (classify.v4 ? PutIpv4Hdr : PutIpv6Hdr)(BUF_TAIL, loc->remoteIP, loc->localIP);
  uint8_t l3addrsLen = classify.v4
                         ? sizeof(struct rte_ipv4_hdr) - offsetof(struct rte_ipv4_hdr, src_addr)
                         : sizeof(struct rte_ipv6_hdr) - offsetof(struct rte_ipv6_hdr, src_addr);
  match->udpOff = match->len;
  match->len += PutUdpHdr(BUF_TAIL, loc->remoteUDP, loc->localUDP);
  match->f = MatchUdp;
  match->l3matchOff = match->udpOff - l3addrsLen;
  match->l3matchLen = l3addrsLen + offsetof(struct rte_udp_hdr, dgram_len);
  if (!classify.vxlan) {
    return;
  }

  match->l3matchLen = l3addrsLen;
  match->len += PutVxlanHdr(BUF_TAIL, loc->vxlan);
  match->len += PutEtherVlanHdr(BUF_TAIL, &loc->innerRemote, &loc->innerLocal, 0, EtherTypeNDN);
  match->f = MatchVxlan;

#undef BUF_TAIL
}

void
EthFlowPattern_Prepare(EthFlowPattern* flow, const EthLocator* loc)
{
  ClassifyResult classify = EthLocator_Classify(loc);

  *flow = (const EthFlowPattern){ 0 };
  size_t i = 0;
#define MASK(field) memset(&(field), 0xFF, sizeof(field))
#define APPEND(typ, field)                                                                         \
  do {                                                                                             \
    flow->pattern[i].type = RTE_FLOW_ITEM_TYPE_##typ;                                              \
    flow->pattern[i].spec = &flow->field##Spec;                                                    \
    flow->pattern[i].mask = &flow->field##Mask;                                                    \
    ++i;                                                                                           \
    NDNDPDK_ASSERT(i < RTE_DIM(flow->pattern));                                                    \
    flow->pattern[i].type = RTE_FLOW_ITEM_TYPE_END;                                                \
  } while (false)

  MASK(flow->ethMask.dst);
  if (classify.multicast) {
    rte_ether_addr_copy(&loc->remote, &flow->ethSpec.dst);
  } else {
    MASK(flow->ethMask.src);
    rte_ether_addr_copy(&loc->local, &flow->ethSpec.dst);
    rte_ether_addr_copy(&loc->remote, &flow->ethSpec.src);
  }
  MASK(flow->ethMask.type);
  flow->ethSpec.type = rte_cpu_to_be_16(classify.etherType);
  APPEND(ETH, eth);

  if (loc->vlan != 0) {
    MASK(flow->vlanMask.tci);
    flow->vlanSpec.tci = rte_cpu_to_be_16(loc->vlan);
    flow->vlanSpec.inner_type = flow->ethSpec.type;
    flow->ethSpec.type = rte_cpu_to_be_16(RTE_ETHER_TYPE_VLAN);
    APPEND(VLAN, vlan);
  }

  if (!classify.udp) {
    MASK(flow->vlanMask.inner_type);
    return;
  }
  flow->pattern[0].spec = NULL; // ETH
  flow->pattern[0].mask = NULL;

  if (classify.v4) {
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

  MASK(flow->udpMask.hdr.dst_port);
  PutUdpHdr((uint8_t*)(&flow->udpSpec.hdr), loc->remoteUDP, loc->localUDP);
  APPEND(UDP, udp);

  if (!classify.vxlan) {
    MASK(flow->udpMask.hdr.src_port);
    return;
  }

  MASK(flow->vxlanMask.vni);
  flow->vxlanSpec.vni[0] = (uint8_t)(loc->vxlan >> 16);
  flow->vxlanSpec.vni[1] = (uint8_t)(loc->vxlan >> 8);
  flow->vxlanSpec.vni[2] = (uint8_t)(loc->vxlan >> 0);
  APPEND(VXLAN, vxlan);

  MASK(flow->innerEthMask.dst);
  MASK(flow->innerEthMask.src);
  MASK(flow->innerEthMask.type);
  rte_ether_addr_copy(&loc->innerLocal, &flow->innerEthSpec.dst);
  rte_ether_addr_copy(&loc->innerRemote, &flow->innerEthSpec.src);
  flow->innerEthSpec.type = rte_cpu_to_be_16(EtherTypeNDN);
  APPEND(ETH, innerEth);

#undef MASK
#undef APPEND
}

__attribute__((nonnull)) static void
TxNoHdr(const EthTxHdr* hdr, struct rte_mbuf* m, bool newBurst)
{}

__attribute__((nonnull)) static __rte_always_inline void
TxPrepend(const EthTxHdr* hdr, struct rte_mbuf* m)
{
  char* room = rte_pktmbuf_prepend(m, hdr->len);
  NDNDPDK_ASSERT(room != NULL); // enough headroom is required
  rte_memcpy(room, hdr->buf, hdr->len);
}

__attribute__((nonnull)) static void
TxEther(const EthTxHdr* hdr, struct rte_mbuf* m, bool newBurst)
{
  TxPrepend(hdr, m);
}

static __rte_always_inline uint16_t
TxMakeVxlanSrcPort(bool newBurst)
{
  RTE_PER_LCORE(txVxlanSrcPort) += (uint16_t)newBurst;
  return (RTE_PER_LCORE(txVxlanSrcPort) & VXLAN_SRCPORT_MASK) | VXLAN_SRCPORT_BASE;
}

__attribute__((nonnull)) static __rte_always_inline struct rte_ipv4_hdr*
TxUdp4(const EthTxHdr* hdr, struct rte_mbuf* m, bool newBurst)
{
  TxPrepend(hdr, m);
  struct rte_ipv4_hdr* ip = rte_pktmbuf_mtod_offset(m, struct rte_ipv4_hdr*, hdr->l2len);
  struct rte_udp_hdr* udp = RTE_PTR_ADD(ip, sizeof(*ip));
  uint16_t ipLen = m->pkt_len - hdr->l2len;
  ip->total_length = rte_cpu_to_be_16(ipLen);
  udp->dgram_len = rte_cpu_to_be_16(ipLen - sizeof(*ip));
  if (hdr->vxlanSrcPort) {
    udp->src_port = rte_cpu_to_be_16(TxMakeVxlanSrcPort(newBurst));
  }
  return ip;
}

__attribute__((nonnull)) static void
TxUdp4Checksum(const EthTxHdr* hdr, struct rte_mbuf* m, bool newBurst)
{
  struct rte_ipv4_hdr* ip = TxUdp4(hdr, m, newBurst);
  ip->hdr_checksum = rte_ipv4_cksum(ip);
}

__attribute__((nonnull)) static void
TxUdp4Offload(const EthTxHdr* hdr, struct rte_mbuf* m, bool newBurst)
{
  struct rte_ipv4_hdr* ip = TxUdp4(hdr, m, newBurst);
  m->l2_len = hdr->l2len;
  m->l3_len = sizeof(*ip);
  m->ol_flags |= PKT_TX_IPV4 | PKT_TX_IP_CKSUM;
}

__attribute__((nonnull)) static __rte_always_inline struct rte_ipv6_hdr*
TxUdp6(const EthTxHdr* hdr, struct rte_mbuf* m, bool newBurst)
{
  TxPrepend(hdr, m);
  struct rte_ipv6_hdr* ip = rte_pktmbuf_mtod_offset(m, struct rte_ipv6_hdr*, hdr->l2len);
  struct rte_udp_hdr* udp = RTE_PTR_ADD(ip, sizeof(*ip));
  ip->payload_len = rte_cpu_to_be_16(m->pkt_len - hdr->l2len - sizeof(*ip));
  udp->dgram_len = ip->payload_len;
  if (hdr->vxlanSrcPort) {
    udp->src_port = rte_cpu_to_be_16(TxMakeVxlanSrcPort(newBurst));
  }
  return ip;
}

__attribute__((nonnull)) static void
TxUdp6Checksum(const EthTxHdr* hdr, struct rte_mbuf* m, bool newBurst)
{
  NDNDPDK_ASSERT(rte_pktmbuf_is_contiguous(m));
  struct rte_ipv6_hdr* ip = TxUdp6(hdr, m, newBurst);
  struct rte_udp_hdr* udp = RTE_PTR_ADD(ip, sizeof(*ip));
  udp->dgram_cksum = rte_ipv6_udptcp_cksum(ip, udp);
}

__attribute__((nonnull)) static void
TxUdp6Offload(const EthTxHdr* hdr, struct rte_mbuf* m, bool newBurst)
{
  struct rte_ipv6_hdr* ip = TxUdp6(hdr, m, newBurst);
  struct rte_udp_hdr* udp = RTE_PTR_ADD(ip, sizeof(*ip));
  m->l2_len = hdr->l2len;
  m->l3_len = sizeof(*ip);
  m->ol_flags |= PKT_TX_IPV6 | PKT_TX_UDP_CKSUM;
  udp->dgram_cksum = rte_ipv6_phdr_cksum(ip, m->ol_flags);
}

void
EthTxHdr_Prepare(EthTxHdr* hdr, const EthLocator* loc, bool hasChecksumOffloads)
{
  ClassifyResult classify = EthLocator_Classify(loc);

  *hdr = (const EthTxHdr){ .f = TxEther };
  if (classify.etherType == 0) {
    hdr->f = TxNoHdr;
    return;
  }

#define BUF_TAIL (RTE_PTR_ADD(hdr->buf, hdr->len))

  hdr->l2len = PutEtherVlanHdr(BUF_TAIL, &loc->local, &loc->remote, loc->vlan, classify.etherType);
  hdr->len += hdr->l2len;

  if (!classify.udp) {
    return;
  }
  hdr->f = classify.v4 ? (hasChecksumOffloads ? TxUdp4Offload : TxUdp4Checksum)
                       : (hasChecksumOffloads ? TxUdp6Offload : TxUdp6Checksum);
  hdr->len += (classify.v4 ? PutIpv4Hdr : PutIpv6Hdr)(BUF_TAIL, loc->localIP, loc->remoteIP);
  hdr->len += PutUdpHdr(BUF_TAIL, loc->localUDP, loc->remoteUDP);

  if (!classify.vxlan) {
    return;
  }
  hdr->vxlanSrcPort = true;
  hdr->len += PutVxlanHdr(BUF_TAIL, loc->vxlan);
  hdr->len += PutEtherVlanHdr(BUF_TAIL, &loc->innerLocal, &loc->innerRemote, 0, EtherTypeNDN);

#undef BUF_TAIL
}
