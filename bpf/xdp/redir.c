/**
 * @file
 * The redir XDP program redirects packets matching face_map to an XSK.
 */
#include "api.h"

struct {
  __uint(type, BPF_MAP_TYPE_XSKMAP);
  __uint(max_entries, 64);
  __type(key, int32_t);
  __type(value, int32_t);
} xsks_map SEC(".maps");

struct {
  __uint(type, BPF_MAP_TYPE_HASH);
  __uint(max_entries, 64);
  __type(key, EthXdpLocator);
  __type(value, int32_t);
} face_map SEC(".maps");

SEC("xdp") int xdp_prog(struct xdp_md* ctx) {
  const void* pkt = (const void*)(long)ctx->data;
  EthXdpLocator loc = {0};

  const struct ethhdr* eth = PacketPtrAs((const struct ethhdr*)pkt, ETH_HLEN);
  pkt += ETH_HLEN;
  if (eth->h_dest[0] & 0x01) {
    memcpy(loc.ether, eth->h_dest, ETH_ALEN);
  } else {
    memcpy(loc.ether, eth->h_dest, 2 * ETH_ALEN);
  }
  uint16_t ethProto = eth->h_proto;
  if (ethProto == bpf_htons(ETH_P_8021Q)) {
    const struct vlanhdr* vlan = PacketPtrAs((const struct vlanhdr*)pkt);
    pkt += sizeof(*vlan);
    loc.vlan = vlan->vlan_tci & bpf_htons(0x0FFF);
    ethProto = vlan->eth_proto;
  }

  uint8_t ipProto = 0;
  switch (ethProto) {
    case bpf_htons(EtherTypeNDN):
      goto FILTER;
    case bpf_htons(ETH_P_IP): {
      const struct iphdr* ipv4 = PacketPtrAs((const struct iphdr*)pkt);
      pkt += sizeof(*ipv4);
      memcpy(loc.ip, &ipv4->saddr, 2 * sizeof(struct in_addr));
      ipProto = ipv4->protocol;
      break;
    }
    case bpf_htons(ETH_P_IPV6): {
      const struct ipv6hdr* ipv6 = PacketPtrAs((const struct ipv6hdr*)pkt);
      pkt += sizeof(*ipv6);
      memcpy(loc.ip, &ipv6->saddr, 2 * sizeof(struct in6_addr));
      ipProto = ipv6->nexthdr;
      break;
    }
    default:
      goto REJECT;
  }

  if (ipProto != IPPROTO_UDP) {
    goto REJECT;
  }
  const struct udphdr* udp = PacketPtrAs((const struct udphdr*)pkt);
  pkt += sizeof(*udp);
  loc.udpSrc = udp->source;
  loc.udpDst = udp->dest;
  switch (udp->dest) {
    case bpf_htons(UDPPortVXLAN): {
      loc.udpSrc = 0;

      const VxlanInnerHdr* vxi = PacketPtrAs((const VxlanInnerHdr*)pkt);
      pkt += sizeof(*vxi);
      enum {
        vxiOffsetVni = offsetof(VxlanInnerHdr, vx.vni),
        vxiOffsetEth = offsetof(VxlanInnerHdr, eth.h_proto),
        vxiLen = vxiOffsetEth - vxiOffsetVni,
        locOffsetVni = offsetof(EthXdpLocator, vx.vni),
        locOffsetEth = offsetof(EthXdpLocator, vx.inner) + sizeof(loc.vx.inner),
        locLen = locOffsetEth - locOffsetVni,
      };
      static_assert(vxiLen == locLen, "");
      memcpy(loc.vx.vni, vxi->vx.vni, locLen); // VNI + innerLocal + innerRemote
      loc.vx.rsvd1 = 0;
      break;
    }
    case bpf_htons(UDPPortGTP): {
      const GtpInnerHdr* gih = PacketPtrAs((const GtpInnerHdr*)pkt);
      pkt += sizeof(*gih);
      if (!EthGtpHdr_IsUplink(&gih->gtp) || gih->ipv4.version != IPVERSION) {
        goto REJECT;
      }
      loc.gtp.teid = gih->gtp.hdr.teid;
      loc.gtp.qfi = gih->gtp.psc.qfi;
      break;
    }
  }
  goto FILTER;

REJECT:
  return XDP_PASS;

FILTER:;
  int32_t* queue = bpf_map_lookup_elem(&face_map, &loc);
  if (queue == NULL) {
    return XDP_PASS;
  }
  return bpf_redirect_map(&xsks_map, *queue, XDP_PASS);
}
