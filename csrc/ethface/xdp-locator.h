#ifndef NDNDPDK_ETHFACE_XDP_LOCATOR_H
#define NDNDPDK_ETHFACE_XDP_LOCATOR_H

/** @file */

#ifdef __BPF__
// disable asm instructions
#define RTE_FORCE_INTRINSICS
#endif

#include "../core/common.h"
#include <rte_gtp.h>

/**
 * @brief EthFace address matcher in XDP program.
 *
 * Unused fields must be zero.
 */
typedef struct EthXdpLocator {
  union {
    struct {
      uint8_t vni[3];       ///< VXLAN Network Identifier (big endian)
      uint8_t rsvd1;        ///< must be zero
      uint8_t inner[2 * 6]; ///< inner Ethernet destination and source
    } vx;
    struct {
      uint32_t teid; ///< GTPv1U Tunnel Endpoint Identifier (big endian)
      uint8_t qfi;   ///< GTPv1U QoS Flow Identifier
    } gtp;
  };
  uint16_t vlan;        ///< VLAN identifier (big endian)
  uint16_t udpSrc;      ///< outer UDP source port (big endian, 0 for VXLAN/GTP)
  uint16_t udpDst;      ///< outer UDP destination port (big endian)
  uint8_t ether[2 * 6]; ///< outer Ethernet destination and source
  uint8_t ip[2 * 16];   ///< outer IPv4/IPv6 source and destination
} __rte_packed EthXdpLocator;

/** @brief Overwritten header after matching in XDP program. */
typedef struct EthXdpHdr {
  uint64_t magic;  ///< UINT64_MAX
  uint32_t fmv;    ///< face_map value
  uint16_t hdrLen; ///< header length
} __rte_packed EthXdpHdr;

/** @brief GTP-U header with PDU session container. */
typedef struct EthGtpHdr {
  struct rte_gtp_hdr hdr;
  struct rte_gtp_hdr_ext_word ext;
  struct rte_gtp_psc_generic_hdr psc;
  uint8_t next;
} __rte_packed EthGtpHdr;

enum {
  /** @brief Value of @c EthGtpHdr.ext.next_ext . */
  EthGtpExtTypePsc = 0x85,
};

/** @brief Determine whether @p gtp is uplink packet with QFI. */
__attribute__((nonnull)) static __rte_always_inline bool
EthGtpHdr_IsUplink(const EthGtpHdr* gtp) {
  return gtp->hdr.e == 1 && gtp->ext.next_ext == EthGtpExtTypePsc && gtp->psc.type == 1;
}

#ifndef __BPF__

typedef struct EthLocator EthLocator;

/** @brief Prepare XDP locator from locator. */
__attribute__((nonnull)) void
EthXdpLocator_Prepare(EthXdpLocator* xl, const EthLocator* loc);

#endif // __BPF__

#endif // NDNDPDK_ETHFACE_XDP_LOCATOR_H
