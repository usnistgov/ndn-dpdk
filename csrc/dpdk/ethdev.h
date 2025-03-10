#ifndef NDNDPDK_DPDK_ETHDEV_H
#define NDNDPDK_DPDK_ETHDEV_H

/** @file */

#include "../core/common.h"
#include <rte_ethdev.h>
#include <rte_flow.h>

/**
 * @brief Bit flags for rte_flow preferences.
 * @see https://doc.dpdk.org/guides/nics/overview.html "rte_flow items availability"
 *
 * Bits  0-23 are to describe hardware capabilities.
 * Bits 24-31 are to describe API level behaviors.
 */
typedef enum EthFlowFlags {
  /**
   * @brief How to generate flow items for pass-through face.
   * 0 = empty pattern.
   * 1 = @c RTE_FLOW_ITEM_TYPE_ETH with EtherType=ARP.
   */
  EthFlowFlagsPassthruArp = RTE_BIT32(0),

  /**
   * @brief How to generate VXLAN flow items.
   * 0 = prefer @c RTE_FLOW_ITEM_TYPE_VXLAN and @c RTE_FLOW_ITEM_TYPE_ETH .
   * 1 = prefer @c RTE_FLOW_ITEM_TYPE_RAW .
   */
  EthFlowFlagsVxRaw = RTE_BIT32(8),

  /**
   * @brief How to generate GTP-U flow item.
   * 0 = prefer @c RTE_FLOW_ITEM_TYPE_GTPU .
   * 1 = prefer @c RTE_FLOW_ITEM_TYPE_GTP .
   */
  EthFlowFlagsGtp = RTE_BIT32(12),

  /**
   * @brief How to generate actions when RSS is used.
   * 0 = keep MARK action.
   * 1 = skip MARK action.
   */
  EthFlowFlagsRssUnmarked = RTE_BIT32(16),

  /**
   * @brief How to generate actions for Ethernet face.
   * 0 = keep MARK action.
   * 1 = skip MARK action.
   */
  EthFlowFlagsEtherUnmarked = RTE_BIT32(17),

  /** @brief Whether the ethdev is flow isolated. */
  EthFlowFlagsIsolated = RTE_BIT32(24),

  /** @brief Whether the flow has MARK action set to FaceID. */
  EthFlowFlagsMarked = RTE_BIT32(25),

  /** @brief Whether rte_flow is unsupported and should not be attempted. */
  EthFlowFlagsDisabled = RTE_BIT32(31),
} __rte_packed EthFlowFlags;
static_assert(sizeof(EthFlowFlags) == sizeof(uint32_t), "");

/** @brief Retrieve whether an Ethernet device is DOWN. */
static inline bool
EthDev_IsDown(uint16_t port) {
  struct rte_eth_link link;
  int res = rte_eth_link_get_nowait(port, &link);
  return res != 0 || link.link_status == RTE_ETH_LINK_DOWN;
}

#endif // NDNDPDK_DPDK_ETHDEV_H
