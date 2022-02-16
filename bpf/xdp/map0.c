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

struct
{
  __uint(type, BPF_MAP_TYPE_XSKMAP);
  __uint(max_entries, 64);
  __type(key, int);
  __type(value, int);
} xsks_map SEC(".maps");

SEC("xdp_sock") int xdp_sock_prog(struct xdp_md* ctx)
{
  const void* pkt = (const void*)(long)ctx->data;

  const struct ethhdr* eth = PacketPtrAs((const struct ethhdr*)pkt);
  pkt += sizeof(*eth);
  uint16_t ethProto = eth->h_proto;
  if (eth->h_proto == bpf_htons(ETH_P_8021Q)) {
    const struct vlanhdr* vlan = PacketPtrAs((const struct vlanhdr*)pkt);
    pkt += sizeof(*vlan);
    ethProto = vlan->eth_proto;
  }

  uint8_t ipProto = 0;
  switch (ethProto) {
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
      goto REJECT;
  }

  if (ipProto != IPPROTO_UDP) {
    goto REJECT;
  }
  const struct udphdr* udp = PacketPtrAs((const struct udphdr*)pkt);
  pkt += sizeof(*udp);
  if (udp->dest == bpf_htons(UDPPortNDN)) {
    goto ACCEPT;
  }

REJECT:
  return XDP_PASS;

ACCEPT:
  return bpf_redirect_map(&xsks_map, 0, XDP_PASS);
}
