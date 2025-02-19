#ifndef NDNDPDK_ETHFACE_HDR_IMPL_H
#define NDNDPDK_ETHFACE_HDR_IMPL_H

/** @file */

#include "../ndni/an.h"
#include "locator.h"

enum {
  IP_HOPLIMIT_VALUE = 64,
  V4_IN_V6_PREFIX_OCTETS = 12,
  GTP_INNER_LEN = sizeof(struct rte_ipv4_hdr) + sizeof(struct rte_udp_hdr),
};

__attribute__((nonnull)) static inline uint8_t
PutEtherHdr(uint8_t* buffer, const struct rte_ether_addr src, const struct rte_ether_addr dst,
            uint16_t vid, uint16_t etherType) {
  struct rte_ether_hdr* ether = (struct rte_ether_hdr*)buffer;
  ether->dst_addr = dst;
  ether->src_addr = src;
  ether->ether_type = rte_cpu_to_be_16(vid == 0 ? etherType : RTE_ETHER_TYPE_VLAN);
  return RTE_ETHER_HDR_LEN;
}

__attribute__((nonnull)) static inline uint8_t
PutVlanHdr(uint8_t* buffer, uint16_t vid, uint16_t etherType) {
  struct rte_vlan_hdr* vlan = (struct rte_vlan_hdr*)buffer;
  vlan->vlan_tci = rte_cpu_to_be_16(vid);
  vlan->eth_proto = rte_cpu_to_be_16(etherType);
  return RTE_VLAN_HLEN;
}

__attribute__((nonnull)) static inline uint8_t
PutEtherVlanHdr(uint8_t* buffer, const struct rte_ether_addr src, const struct rte_ether_addr dst,
                uint16_t vid, uint16_t etherType) {
  uint8_t off = PutEtherHdr(buffer, src, dst, vid, etherType);
  if (vid != 0) {
    off += PutVlanHdr(RTE_PTR_ADD(buffer, off), vid, etherType);
  }
  return off;
}

__attribute__((nonnull)) static inline uint8_t
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

__attribute__((nonnull)) static inline uint8_t
PutIpv6Hdr(uint8_t* buffer, const struct rte_ipv6_addr src, const struct rte_ipv6_addr dst) {
  struct rte_ipv6_hdr* ip = (struct rte_ipv6_hdr*)buffer;
  ip->vtc_flow = rte_cpu_to_be_32(6 << 28); // IP version 6
  ip->proto = IPPROTO_UDP;
  ip->hop_limits = IP_HOPLIMIT_VALUE;
  ip->src_addr = src;
  ip->dst_addr = dst;
  return sizeof(*ip);
}

__attribute__((nonnull)) static inline uint16_t
PutUdpHdr(uint8_t* buffer, uint16_t src, uint16_t dst) {
  struct rte_udp_hdr* udp = (struct rte_udp_hdr*)buffer;
  udp->src_port = rte_cpu_to_be_16(src);
  udp->dst_port = rte_cpu_to_be_16(dst);
  return sizeof(*udp);
}

__attribute__((nonnull)) static __rte_always_inline void
PutVxlanVni(uint8_t vni[3], uint32_t vniH) {
  rte_be32_t vniB = rte_cpu_to_be_32(vniH);
  memcpy(vni, RTE_PTR_ADD(&vniB, 1), 3);
}

__attribute__((nonnull)) static inline uint8_t
PutVxlanHdr(uint8_t* buffer, uint32_t vni) {
  struct rte_vxlan_hdr* vxlan = (struct rte_vxlan_hdr*)buffer;
  vxlan->flag_i = 1;
  PutVxlanVni(vxlan->vni, vni);
  return sizeof(*vxlan);
}

__attribute__((nonnull)) static inline void
PutGtpHdrMinimal(struct rte_gtp_hdr* hdr, uint32_t teid) {
  hdr->ver = 1;
  hdr->pt = 1;
  hdr->e = 1;
  hdr->msg_type = 0xFF;
  hdr->teid = rte_cpu_to_be_32(teid);
}

__attribute__((nonnull)) static inline uint8_t
PutGtpHdr(uint8_t* buffer, bool ul, uint32_t teid, uint8_t qfi) {
  EthGtpHdr* gtp = (EthGtpHdr*)buffer;
  static_assert(sizeof(*gtp) == 16, "");
  PutGtpHdrMinimal(&gtp->hdr, teid);
  gtp->ext.next_ext = EthGtpExtTypePsc;
  gtp->psc.ext_hdr_len = 1;
  gtp->psc.type = (int)ul;
  gtp->psc.qfi = qfi;
  return sizeof(*gtp);
}

#endif // NDNDPDK_ETHFACE_HDR_IMPL_H
