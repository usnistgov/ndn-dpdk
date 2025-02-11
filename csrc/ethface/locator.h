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

typedef struct EthRxMatch EthRxMatch;

/** @brief EthFace RX matcher. */
struct EthRxMatch {
  bool (*f)(const EthRxMatch* match, const struct rte_mbuf* m);
  uint8_t len;
  uint8_t l2len;
  uint8_t l3matchOff;
  uint8_t l3matchLen;
  uint8_t udpOff;
  uint8_t buf[EthFace_HdrMax];
};

/** @brief Prepare RX matcher from locator. */
__attribute__((nonnull)) void
EthRxMatch_Prepare(EthRxMatch* match, const EthLocator* loc);

/**
 * @brief Determine whether a received frame matches the locator.
 * @param match EthRxMatch prepared by @c EthRxMatch_Prepare .
 */
__attribute__((nonnull)) static inline bool
EthRxMatch_Match(const EthRxMatch* match, const struct rte_mbuf* m) {
  return m->data_len >= match->len && match->f(match, m);
}

/** @brief Prepare XDP locator from locator. */
__attribute__((nonnull)) void
EthXdpLocator_Prepare(EthXdpLocator* xl, const EthLocator* loc);

/** @brief EthFace rte_flow pattern. */
typedef struct EthFlowPattern {
  struct rte_flow_item pattern[7];
  struct rte_flow_item_eth ethSpec;
  struct rte_flow_item_eth ethMask;
  struct rte_flow_item_vlan vlanSpec;
  struct rte_flow_item_vlan vlanMask;
  union {
    struct {
      struct rte_flow_item_ipv4 ip4Spec;
      struct rte_flow_item_ipv4 ip4Mask;
    };
    struct {
      struct rte_flow_item_ipv6 ip6Spec;
      struct rte_flow_item_ipv6 ip6Mask;
    };
  };
  struct rte_flow_item_udp udpSpec;
  struct rte_flow_item_udp udpMask;
  union {
    struct {
      struct rte_flow_item_vxlan vxlanSpec;
      struct rte_flow_item_vxlan vxlanMask;
      struct rte_flow_item_eth innerEthSpec;
      struct rte_flow_item_eth innerEthMask;
    };
    struct {
      struct rte_flow_item_gtp gtpSpec;
      struct rte_flow_item_gtp gtpMask;
    };
  };
} EthFlowPattern;

/**
 * @brief Prepare rte_flow pattern from locator.
 * @param[out] flow Flow pattern.
 * @param[out] priority Flow priority.
 * @param loc Locator.
 * @param prefersFlowItemGTP For GTP-U flow item, whether to use RTE_FLOW_ITEM_TYPE_GTP instead
 *                           of RTE_FLOW_ITEM_TYPE_GTPU. Correct value depends on the NIC driver.
 */
__attribute__((nonnull)) void
EthFlowPattern_Prepare(EthFlowPattern* flow, uint32_t* priority, const EthLocator* loc,
                       bool prefersFlowItemGTP);

typedef struct EthTxHdr EthTxHdr;

/** @brief EthFace TX header template. */
struct EthTxHdr {
  void (*f)(const EthTxHdr* hdr, struct rte_mbuf* m, bool newBurst);
  uint8_t len;
  uint8_t l2len;
  char tunnel;
  uint8_t buf[EthFace_HdrMax];
};

/** @brief Prepare TX header from locator. */
__attribute__((nonnull)) void
EthTxHdr_Prepare(EthTxHdr* hdr, const EthLocator* loc, bool hasChecksumOffloads);

/**
 * @brief Prepend TX header.
 * @param hdr prepared by @c EthTxHdr_Prepare .
 * @param newBurst whether @p m is the first frame in a new burst.
 */
__attribute__((nonnull)) static inline void
EthTxHdr_Prepend(const EthTxHdr* hdr, struct rte_mbuf* m, bool newBurst) {
  hdr->f(hdr, m, newBurst);
}

#endif // NDNDPDK_ETHFACE_LOCATOR_H
