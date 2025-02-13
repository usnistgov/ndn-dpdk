#include "gtpip-table.h"
#include "../core/logger.h"
#include "face.h"

N_LOG_INIT(GtpipTable);

enum {
  // Length of inner IPv4+UDP headers for NDN over GTP-U packets as prepared in EthTxHdr.
  // Since we are dealing with IP traffic that already has IP headers, these headers shall
  // not be prepended onto downlink packets.
  EthTxHdr_InnerLen = sizeof(struct rte_ipv4_hdr) + sizeof(struct rte_udp_hdr),
};

__attribute__((nonnull)) static inline void
PutDownlinkHeader(void* room, EthTxHdr* hdr, uint16_t innerLen) {
  rte_memcpy(room, hdr->buf, hdr->len - EthTxHdr_InnerLen);

  struct rte_ether_hdr* eth = room;
  // outer IPv6 is not implemented
  NDNDPDK_ASSERT(eth->ether_type == rte_cpu_to_be_16(RTE_ETHER_TYPE_IPV4));
  struct rte_ipv4_hdr* ip4 = RTE_PTR_ADD(eth, RTE_ETHER_HDR_LEN);
  struct rte_udp_hdr* udp = RTE_PTR_ADD(ip4, sizeof(*ip4));
  EthGtpHdr* gtp = RTE_PTR_ADD(udp, sizeof(*udp));

  gtp->hdr.plen = rte_cpu_to_be_16(sizeof(*gtp) - sizeof(gtp->hdr) + innerLen);
  udp->dgram_len = rte_cpu_to_be_16(sizeof(*udp) + sizeof(*gtp) + innerLen);
  ip4->total_length = rte_cpu_to_be_16(sizeof(*ip4) + sizeof(*udp) + sizeof(*gtp) + innerLen);
  ip4->hdr_checksum = rte_ipv4_cksum(ip4); // TODO offload
}

bool
GtpipTable_ProcessDownlink(GtpipTable* table, struct rte_mbuf* m) {
  if (unlikely(m->data_len < RTE_ETHER_HDR_LEN + sizeof(struct rte_ipv4_hdr))) {
    return false;
  }
  const struct rte_ether_hdr* eth = rte_pktmbuf_mtod(m, const struct rte_ether_hdr*);
  const struct rte_ipv4_hdr* ip4 =
    rte_pktmbuf_mtod_offset(m, const struct rte_ipv4_hdr*, RTE_ETHER_HDR_LEN);
  if (eth->ether_type != rte_cpu_to_be_16(RTE_ETHER_TYPE_IPV4)) {
    return false;
  }
  rte_be32_t ueIP = ip4->dst_addr;

  void* hdata = NULL;
  int res = rte_hash_lookup_data(table->ipv4, &ueIP, &hdata);
  if (res < 0) {
    return false;
  }

  FaceID id = (FaceID)(uintptr_t)hdata;
  EthFacePriv* priv = Face_GetPriv(Face_Get(id));
  EthTxHdr* hdr = &priv->txHdr;
  NDNDPDK_ASSERT(hdr->tunnel == 'G');

  // strip the Ethernet header, then prepend outer Ethernet+IP+UDP+GTP headers
  uint16_t innerLen = (uint16_t)m->pkt_len - RTE_ETHER_HDR_LEN;
  char* room = rte_pktmbuf_prepend(m, hdr->len - EthTxHdr_InnerLen - RTE_ETHER_HDR_LEN);
  if (unlikely(room == NULL)) {
    return false;
  }
  PutDownlinkHeader(room, hdr, innerLen);

  return true;
}

bool
GtpipTable_ProcessUplink(GtpipTable* table, struct rte_mbuf* pkt) {
  return false;
}
