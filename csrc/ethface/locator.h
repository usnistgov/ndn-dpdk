#ifndef NDNDPDK_ETHFACE_HDR_H
#define NDNDPDK_ETHFACE_HDR_H

/** @file */

#include "../core/common.h"
#include <rte_flow.h>
#include <rte_vxlan.h>

/** @brief EthFace address information. */
typedef struct EthLocator
{
  struct rte_ether_addr local;
  struct rte_ether_addr remote;
  uint16_t vlan;
} EthLocator;

/** @brief EthFace header buffer length. */
#define ETHHDR_BUFLEN                                                                              \
  (RTE_ETHER_HDR_LEN + sizeof(struct rte_vlan_hdr) + sizeof(struct rte_ipv6_hdr) +                 \
   sizeof(struct rte_udp_hdr) + sizeof(struct rte_vxlan_hdr) + RTE_ETHER_HDR_LEN)
static_assert(sizeof(struct rte_ipv4_hdr) <= sizeof(struct rte_ipv6_hdr), "");
static_assert(ETHHDR_BUFLEN <= RTE_PKTMBUF_HEADROOM, "");

/**
 * @brief EthFace RX match function.
 * @param buffer a buffer prepared by @c EthLocator_MakeRxMatch .
 * @param m Ethernet frame.
 * @return whether this frame matches the EthLocator passed to @c EthLocator_MakeRxMatch .
 */
typedef bool (*EthRxMatch)(const uint8_t* buffer, const struct rte_mbuf* m);

/**
 * @brief Create RX match function.
 * @param[out] buffer a buffer of ETHHDR_BUFLEN capacity.
 */
EthRxMatch
EthLocator_MakeRxMatch(const EthLocator* loc, uint8_t* buffer);

/** @brief rte_flow pattern and items derived from EthLocator. */
typedef struct EthFlowPattern
{
  struct rte_flow_item pattern[3];
  struct rte_flow_item_eth ethSpec;
  struct rte_flow_item_eth ethMask;
  struct rte_flow_item_vlan vlanSpec;
  struct rte_flow_item_vlan vlanMask;
} EthFlowPattern;

/** @brief Prepare rte_flow pattern. */
void
EthLocator_MakeFlowPattern(const EthLocator* loc, EthFlowPattern* flow);

/**
 * @brief Create TX header.
 * @param[out] hdr the header to be prepended, a buffer of ETHHDR_BUFLEN capacity.
 * @return header length (for both RX and TX).
 */
uint16_t
EthLocator_MakeTxHdr(const EthLocator* loc, uint8_t* hdr);

#endif // NDNDPDK_ETHFACE_ETH_FACE_H
