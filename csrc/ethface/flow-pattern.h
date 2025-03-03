#ifndef NDNDPDK_ETHFACE_FLOW_PATTERN_H
#define NDNDPDK_ETHFACE_FLOW_PATTERN_H

/** @file */

#include "locator.h"

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
      struct rte_flow_item_raw rawSpec;
      struct rte_flow_item_raw rawMask;
      uint8_t rawSpecBuf[16];
      uint8_t rawMaskBuf[16];
    };
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
 * @param flowFlags @p EthFlowFlags bits.
 */
__attribute__((nonnull)) void
EthFlowPattern_Prepare(EthFlowPattern* flow, uint32_t* priority, const EthLocator* loc,
                       uint32_t flowFlags);

#endif // NDNDPDK_ETHFACE_FLOW_PATTERN_H
