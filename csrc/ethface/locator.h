#ifndef NDNDPDK_ETHFACE_LOCATOR_H
#define NDNDPDK_ETHFACE_LOCATOR_H

/** @file */

#include "../dpdk/ethdev.h"
#include "xdp-locator.h"

/** @brief EthFace address information. */
typedef struct EthLocator {
  struct rte_ether_addr local;
  struct rte_ether_addr remote;
  uint16_t vlan;

  struct rte_ipv6_addr localIP;
  struct rte_ipv6_addr remoteIP;
  uint16_t localUDP;
  uint16_t remoteUDP;

  uint32_t vxlan;
  struct rte_ether_addr innerLocal;
  struct rte_ether_addr innerRemote;

  uint32_t ulTEID;
  uint32_t dlTEID;
  struct rte_ipv6_addr innerLocalIP;
  struct rte_ipv6_addr innerRemoteIP;
  uint8_t ulQFI;
  uint8_t dlQFI;
  bool isGtp;
} EthLocator;

/** @brief Determine whether two locators can coexist on the same port. */
__attribute__((nonnull)) bool
EthLocator_CanCoexist(const EthLocator* a, const EthLocator* b);

typedef struct EthLocatorClass {
  uint16_t etherType; ///< outer EtherType, 0 for memif
  bool passthru;      ///< is passthru face
  bool multicast;     ///< is outer Ethernet multicast?
  bool v4;            ///< is IPv4?
  bool udp;           ///< is UDP?
  char tunnel;        ///< 'V' for VXLAN, 'G' for GTP-U, 0 otherwise
} EthLocatorClass;

/** @brief Classify EthFace locator. */
__attribute__((nonnull)) EthLocatorClass
EthLocator_Classify(const EthLocator* loc);

enum {
  /** @brief EthFace header buffer length. */
  EthFace_HdrMax =
    RTE_ETHER_HDR_LEN + RTE_VLAN_HLEN +
    spdk_max(sizeof(struct rte_ipv4_hdr), sizeof(struct rte_ipv6_hdr)) +
    sizeof(struct rte_udp_hdr) +
    spdk_max(sizeof(struct rte_vxlan_hdr) + RTE_ETHER_HDR_LEN,
             sizeof(EthGtpHdr) + sizeof(struct rte_ipv4_hdr) + sizeof(struct rte_udp_hdr))
};
static_assert(EthFace_HdrMax <= RTE_PKTMBUF_HEADROOM, "");

#endif // NDNDPDK_ETHFACE_LOCATOR_H
