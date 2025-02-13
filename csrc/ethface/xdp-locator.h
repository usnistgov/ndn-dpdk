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
    uint32_t vxlan; ///< VXLAN Network Identifier (big endian)
    uint32_t teid;  ///< GTPv1U Tunnel Endpoint Identifier (big endian)
  };
  uint16_t vlan;        ///< VLAN identifier (big endian)
  uint16_t udpSrc;      ///< UDP source port (big endian, 0 for VXLAN)
  uint16_t udpDst;      ///< UDP destination port (big endian)
  uint8_t ether[2 * 6]; ///< outer Ethernet destination and source
  uint8_t ip[2 * 16];   ///< IPv4/IPv6 source and destination
  union {
    uint8_t inner[2 * 6]; ///< inner Ethernet destination and source
    uint8_t qfi;          ///< GTPv1U QoS Flow Identifier (big endian)
  };
} __rte_packed EthXdpLocator;

/** @brief GTP-U header with PDU session container. */
typedef struct EthGtpHdr {
  struct rte_gtp_hdr hdr;
  struct rte_gtp_hdr_ext_word ext;
  struct rte_gtp_psc_generic_hdr psc;
  uint8_t next;
} __rte_packed EthGtpHdr;

#ifndef __BPF__

typedef struct EthLocator EthLocator;

/** @brief Prepare XDP locator from locator. */
__attribute__((nonnull)) void
EthXdpLocator_Prepare(EthXdpLocator* xl, const EthLocator* loc);

#endif // __BPF__

#endif // NDNDPDK_ETHFACE_XDP_LOCATOR_H
