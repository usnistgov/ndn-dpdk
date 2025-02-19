#ifndef NDNDPDK_BPF_XDP_API_H
#define NDNDPDK_BPF_XDP_API_H

/** @file */

#include "../../csrc/ethface/xdp-locator.h"
#include "../../csrc/ndni/an.h"

#include <linux/bpf.h>
#include <linux/if_ether.h>
#include <linux/in.h>
#include <linux/ip.h>
#include <linux/ipv6.h>
#include <linux/udp.h>

#include <bpf/bpf_endian.h>
#include <bpf/bpf_helpers.h>

struct vlanhdr {
  uint16_t vlan_tci;
  uint16_t eth_proto;
} __rte_packed;

struct vxlanhdr {
  uint8_t flags;
  uint8_t rsvd0[3];
  uint8_t vni[3];
  uint8_t rsvd1;
} __rte_packed;

typedef struct VxlanInnerHdr {
  struct vxlanhdr vx;
  struct ethhdr eth;
  struct udphdr udp;
} __rte_packed VxlanInnerHdr;

enum {
  UDPPortVXLAN = 4789,
  UDPPortGTP = 2152,
};

#define PacketPtrAs_(ptr, size, ...)                                                               \
  __extension__({                                                                                  \
    if ((const uint8_t*)ptr + (size_t)(size) > (const uint8_t*)(long)ctx->data_end) {              \
      return XDP_DROP;                                                                             \
    }                                                                                              \
    pkt;                                                                                           \
  })

/**
 * @brief Perform bounds-checking on packet pointer.
 *
 * This can be used within an XDP program, where `struct xdp_md* ctx` is declared.
 * If the structure dereferenced from the given pointer is within the bounds of the packet,
 * this returns the pointer; otherwise, the packet is dropped.
 *
 * @code
 * const Header* hdr = PacketPtrAs((const Header*)pkt);
 * const Header* hdr = PacketPtrAs((const Header*)pkt, HDR_LEN);
 * @endcode
 */
#define PacketPtrAs(ptr, ...) PacketPtrAs_((ptr), ##__VA_ARGS__, sizeof(*(ptr)))

#endif // NDNDPDK_BPF_XDP_API_H
