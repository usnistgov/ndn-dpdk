#ifndef NDNDPDK_ETHFACE_FLOWDEF_H
#define NDNDPDK_ETHFACE_FLOWDEF_H

/** @file */

#include "../iface/enum.h"
#include "locator.h"

enum {
  EthFlowDef_MaxVariant = 8,
  EthFlowDef_NPatterns = 7,
};

/** @brief EthFace rte_flow pattern. */
typedef struct EthFlowDef {
  struct rte_flow_attr attr;

  struct rte_flow_item pattern[EthFlowDef_NPatterns];
  uint16_t patternSpecLen[EthFlowDef_NPatterns];
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
  union {
    struct rte_flow_action_queue queueAct;
    struct {
      struct rte_flow_action_rss rssAct;
      uint16_t rssQueues[MaxFaceRxThreads];
    };
  };
  struct rte_flow_action_mark markAct;
} EthFlowDef;

typedef enum EthFlowDefResult {
  EthFlowDefResultValid = RTE_BIT32(0),
  EthFlowDefResultMarked = RTE_BIT32(1),
} __rte_packed EthFlowDefResult;

/**
 * @brief Prepare rte_flow definition from locator.
 * @param[out] flow Flow definition.
 * @param loc Locator.
 * @param variant Variant index within [0:EthFlowDef_MaxVariant).
 * @param mark FDIR mark ID.
 * @param queues Dispatched queues.
 */
__attribute__((nonnull)) EthFlowDefResult
EthFlowDef_Prepare(EthFlowDef* flow, const EthLocator* loc, int variant, uint32_t mark,
                   const uint16_t queues[], int nQueues);

__attribute__((nonnull)) void
EthFlowDef_DebugPrint(const EthFlowDef* flow, const char* msg);

/** @brief Update @c error->cause to be an offset if it's within @p flow . */
__attribute__((nonnull)) void
EthFlowDef_UpdateError(const EthFlowDef* flow, struct rte_flow_error* error);

#endif // NDNDPDK_ETHFACE_FLOWDEF_H
