#ifndef NDNDPDK_ETHFACE_FLOWDEF_H
#define NDNDPDK_ETHFACE_FLOWDEF_H

/** @file */

#include "../iface/enum.h"
#include "locator.h"

/** @brief EthFace rte_flow pattern. */
typedef struct EthFlowDef {
  struct rte_flow_attr attr;

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

  struct rte_flow_action actions[3];
  struct rte_flow_action_queue queueAct;
  struct rte_flow_action_rss rssAct;
  uint16_t rssQueues[MaxFaceRxThreads];
  struct rte_flow_action_mark markAct;
} EthFlowDef;

/**
 * @brief Prepare rte_flow definition from locator.
 * @param[out] flow Flow definition.
 * @param loc Locator.
 * @param flowFlags Driver-specific preferences.
 * @param mark FDIR mark ID.
 * @param queues Dispatched queues.
 */
__attribute__((nonnull)) void
EthFlowDef_Prepare(EthFlowDef* flow, const EthLocator* loc, EthFlowFlags flowFlags, uint32_t mark,
                   const uint16_t queues[], int nQueues);

/** @brief Update @c error->cause to be an offset if it's within @p flow . */
__attribute__((nonnull)) void
EthFlowDef_UpdateError(const EthFlowDef* flow, struct rte_flow_error* error);

#endif // NDNDPDK_ETHFACE_FLOWDEF_H
