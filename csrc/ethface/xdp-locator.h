#ifndef NDNDPDK_ETHFACE_XDP_LOCATOR_H
#define NDNDPDK_ETHFACE_XDP_LOCATOR_H

/** @file */

#include "../core/common.h"

/**
 * @brief EthFace address matcher in XDP program.
 *
 * Unused fields must be zero.
 */
typedef struct EthXdpLocator
{
  uint32_t vxlan;       ///< VXLAN Network Identifier (big endian)
  uint16_t vlan;        ///< VLAN identifier (big endian)
  uint16_t udpSrc;      ///< UDP source port (big endian, 0 for VXLAN)
  uint16_t udpDst;      ///< UDP destination port (big endian)
  uint8_t ether[2 * 6]; ///< outer Ethernet destination and source
  uint8_t inner[2 * 6]; ///< inner Ethernet destination and source
  uint8_t ip[2 * 16];   ///< IPv4/IPv6 source and destination
} __rte_packed EthXdpLocator;

#endif // NDNDPDK_ETHFACE_XDP_LOCATOR_H
