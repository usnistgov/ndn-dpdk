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
  if (eth->h_proto == bpf_htons(NDN_ETHERTYPE)) {
    return bpf_redirect_map(&xsks_map, 0, XDP_PASS);
  }

  return XDP_PASS;
}
