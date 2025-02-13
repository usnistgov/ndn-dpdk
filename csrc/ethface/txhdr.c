#include "txhdr.h"
#include "hdr-impl.h"

enum {
  VXLAN_SRCPORT_BASE = 0xC000,
  VXLAN_SRCPORT_MASK = 0x3FFF,
};

static RTE_LCORE_VAR_HANDLE(uint16_t, txVxlanSrcPort);
RTE_LCORE_VAR_INIT(txVxlanSrcPort)

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
