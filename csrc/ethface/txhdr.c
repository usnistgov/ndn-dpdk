#include "txhdr.h"
#include "hdr-impl.h"

enum {
  VXLAN_SRCPORT_BASE = 0xC000,
  VXLAN_SRCPORT_MASK = 0x3FFF,
  GTPIP_INNERLEN = sizeof(struct rte_ipv4_hdr) + sizeof(struct rte_udp_hdr),
};

static RTE_LCORE_VAR_HANDLE(uint16_t, txVxlanSrcPort);
RTE_LCORE_VAR_INIT(txVxlanSrcPort)

__attribute__((nonnull)) static void
TxNoHdr(const EthTxHdr* hdr, struct rte_mbuf* m, EthTxHdrFlags flags) {}

__attribute__((nonnull)) static __rte_always_inline void
TxPrepend(const EthTxHdr* hdr, struct rte_mbuf* m, EthTxHdrFlags flags) {
  uint8_t copyLen = hdr->len;
  uint8_t prependLen = hdr->len;
  if (unlikely((flags & EthTxHdrFlagsGtpip) != 0)) {
    NDNDPDK_ASSERT(hdr->tunnel == 'G');
    copyLen -= GTPIP_INNERLEN;                        // don't prepend inner IPv4 and UDP headers
    prependLen -= GTPIP_INNERLEN + RTE_ETHER_HDR_LEN; // overwrite existing Ethernet header
  }
  char* room = rte_pktmbuf_prepend(m, prependLen);
  NDNDPDK_ASSERT(room != NULL); // enough headroom is required
  rte_memcpy(room, hdr->buf, copyLen);
}

__attribute__((nonnull)) static void
TxEther(const EthTxHdr* hdr, struct rte_mbuf* m, EthTxHdrFlags flags) {
  TxPrepend(hdr, m, 0);
}

__attribute__((nonnull)) static __rte_always_inline void
TxUdpCommon(const EthTxHdr* hdr, struct rte_udp_hdr* udp, uint16_t udpLen, EthTxHdrFlags flags) {
  udp->dgram_len = rte_cpu_to_be_16(udpLen);
  switch (hdr->tunnel) {
    case 'V': {
      static_assert((VXLAN_SRCPORT_BASE & VXLAN_SRCPORT_MASK) == 0, "");
      uint16_t incPort = (flags & EthTxHdrFlagsNewBurst) != 0 ? 1 : 0;
      uint16_t srcPort = (*LCORE_VAR_SAFE(txVxlanSrcPort) += incPort);
      udp->src_port = rte_cpu_to_be_16((srcPort & VXLAN_SRCPORT_MASK) | VXLAN_SRCPORT_BASE);
      break;
    }
    case 'G': {
      uint16_t gtpLen = udpLen - sizeof(*udp);
      EthGtpHdr* gtp = RTE_PTR_ADD(udp, sizeof(*udp));
      gtp->hdr.plen = rte_cpu_to_be_16(gtpLen - sizeof(gtp->hdr));
      if (likely((flags & EthTxHdrFlagsGtpip) == 0)) {
        struct rte_ipv4_hdr* iip = RTE_PTR_ADD(gtp, sizeof(*gtp));
        struct rte_udp_hdr* iudp = RTE_PTR_ADD(iip, sizeof(*iip));
        iip->total_length = rte_cpu_to_be_16(gtpLen - sizeof(*gtp));
        iudp->dgram_len = rte_cpu_to_be_16(gtpLen - sizeof(*gtp) - sizeof(*iip));
        iip->hdr_checksum = rte_ipv4_cksum(iip);
      }
      break;
    }
  }
}

__attribute__((nonnull)) static __rte_always_inline struct rte_ipv4_hdr*
TxUdp4(const EthTxHdr* hdr, struct rte_mbuf* m, EthTxHdrFlags flags) {
  TxPrepend(hdr, m, flags);
  struct rte_ipv4_hdr* ip = rte_pktmbuf_mtod_offset(m, struct rte_ipv4_hdr*, hdr->l2len);
  struct rte_udp_hdr* udp = RTE_PTR_ADD(ip, sizeof(*ip));
  uint16_t ipLen = m->pkt_len - hdr->l2len;
  ip->total_length = rte_cpu_to_be_16(ipLen);
  TxUdpCommon(hdr, udp, ipLen - sizeof(*ip), flags);
  return ip;
}

__attribute__((nonnull)) static void
TxUdp4Checksum(const EthTxHdr* hdr, struct rte_mbuf* m, EthTxHdrFlags flags) {
  struct rte_ipv4_hdr* ip = TxUdp4(hdr, m, flags);
  ip->hdr_checksum = rte_ipv4_cksum(ip);
}

__attribute__((nonnull)) static void
TxUdp4Offload(const EthTxHdr* hdr, struct rte_mbuf* m, EthTxHdrFlags flags) {
  struct rte_ipv4_hdr* ip = TxUdp4(hdr, m, flags);
  m->l2_len = hdr->l2len;
  m->l3_len = sizeof(*ip);
  m->ol_flags |= RTE_MBUF_F_TX_IPV4 | RTE_MBUF_F_TX_IP_CKSUM;
}

__attribute__((nonnull)) static __rte_always_inline struct rte_ipv6_hdr*
TxUdp6(const EthTxHdr* hdr, struct rte_mbuf* m, EthTxHdrFlags flags) {
  TxPrepend(hdr, m, flags);
  struct rte_ipv6_hdr* ip = rte_pktmbuf_mtod_offset(m, struct rte_ipv6_hdr*, hdr->l2len);
  struct rte_udp_hdr* udp = RTE_PTR_ADD(ip, sizeof(*ip));
  TxUdpCommon(hdr, udp, m->pkt_len - hdr->l2len - sizeof(*ip), flags);
  ip->payload_len = udp->dgram_len;
  return ip;
}

__attribute__((nonnull)) static void
TxUdp6Checksum(const EthTxHdr* hdr, struct rte_mbuf* m, EthTxHdrFlags flags) {
  struct rte_ipv6_hdr* ip = TxUdp6(hdr, m, flags);
  struct rte_udp_hdr* udp = RTE_PTR_ADD(ip, sizeof(*ip));
  udp->dgram_cksum = rte_ipv6_udptcp_cksum_mbuf(m, ip, hdr->l2len + sizeof(*ip));
}

__attribute__((nonnull)) static void
TxUdp6Offload(const EthTxHdr* hdr, struct rte_mbuf* m, EthTxHdrFlags flags) {
  struct rte_ipv6_hdr* ip = TxUdp6(hdr, m, flags);
  struct rte_udp_hdr* udp = RTE_PTR_ADD(ip, sizeof(*ip));
  m->l2_len = hdr->l2len;
  m->l3_len = sizeof(*ip);
  m->ol_flags |= RTE_MBUF_F_TX_IPV6 | RTE_MBUF_F_TX_UDP_CKSUM;
  udp->dgram_cksum = rte_ipv6_phdr_cksum(ip, m->ol_flags);
}

const EthTxHdr_PrependFunc EthTxHdr_PrependJmp[] = {
  [EthTxHdrActNoHdr] = TxNoHdr,
  [EthTxHdrActEther] = TxEther,
  [EthTxHdrActUdp4Checksum] = TxUdp4Checksum,
  [EthTxHdrActUdp4Offload] = TxUdp4Offload,
  [EthTxHdrActUdp6Checksum] = TxUdp6Checksum,
  [EthTxHdrActUdp6Offload] = TxUdp6Offload,
};

void
EthTxHdr_Prepare(EthTxHdr* hdr, const EthLocator* loc, bool hasChecksumOffloads) {
  EthLocatorClass c = EthLocator_Classify(loc);

  *hdr = (const EthTxHdr){0};
  if (c.etherType == 0) {
    hdr->act = EthTxHdrActNoHdr;
    return;
  }

#define BUF_TAIL (RTE_PTR_ADD(hdr->buf, hdr->len))

  hdr->l2len = PutEtherVlanHdr(BUF_TAIL, &loc->local, &loc->remote, loc->vlan, c.etherType);
  hdr->len += hdr->l2len;
  if (!c.udp) {
    hdr->act = EthTxHdrActEther;
    return;
  }

  hdr->act = 0b1000 | (c.v4 ? 0b10 : 0b00) | (hasChecksumOffloads ? 0b1 : 0b0);
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
