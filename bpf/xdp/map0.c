/**
 * @file
 * The map0 XDP program redirects all NDN packets to XDP socket #0.
 *
 * It understands these transports:
 * @li NDN over Ethernet.
 * @li NDN over IPv4 + UDP.
 * @li NDN over IPv6 + UDP.
 */
#include "api.h"

struct bpf_map_def SEC("maps") xsks_map = {
  .type = BPF_MAP_TYPE_XSKMAP,
  .key_size = sizeof(int),
  .value_size = sizeof(int),
  .max_entries = 64,
};

SEC("xdp_sock") int xdp_sock_prog(struct xdp_md* ctx)
{
  const void* pkt = (const void*)(long)ctx->data;

  const struct ethhdr* eth = PacketPtrAs((const struct ethhdr*)pkt);
  pkt += sizeof(*eth);

  uint8_t ipProto = 0;
  switch (eth->h_proto) {
    case bpf_htons(EtherTypeNDN):
      goto ACCEPT;
    case bpf_htons(ETH_P_IP): {
      const struct iphdr* ipv4 = PacketPtrAs((const struct iphdr*)pkt);
      pkt += sizeof(*ipv4);
      ipProto = ipv4->protocol;
      break;
    }
    case bpf_htons(ETH_P_IPV6): {
      const struct ipv6hdr* ipv6 = PacketPtrAs((const struct ipv6hdr*)pkt);
      pkt += sizeof(*ipv6);
      ipProto = ipv6->nexthdr;
      break;
    }
    default:
      return XDP_PASS;
  }

  if (ipProto != IPPROTO_UDP) {
    return XDP_PASS;
  }
  const struct udphdr* udp = PacketPtrAs((const struct udphdr*)pkt);
  pkt += sizeof(*udp);
  if (udp->dest != bpf_htons(UDPPortNDN)) {
    return XDP_PASS;
  }

ACCEPT:
  return bpf_redirect_map(&xsks_map, 0, XDP_PASS);
}
