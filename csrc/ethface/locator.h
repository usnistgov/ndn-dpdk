#ifndef NDNDPDK_ETHFACE_LOCATOR_H
#define NDNDPDK_ETHFACE_LOCATOR_H

/** @file */

#include "../core/common.h"
#include <rte_flow.h>
#include <rte_vxlan.h>

/** @brief EthFace header buffer length. */
#define ETHHDR_MAXLEN                                                                              \
  (RTE_ETHER_HDR_LEN + sizeof(struct rte_vlan_hdr) + sizeof(struct rte_ipv6_hdr) +                 \
   RTE_ETHER_VXLAN_HLEN + RTE_ETHER_HDR_LEN)
static_assert(sizeof(struct rte_ipv4_hdr) <= sizeof(struct rte_ipv6_hdr), "");
static_assert(ETHHDR_MAXLEN <= RTE_PKTMBUF_HEADROOM, "");

/** @brief EthFace address information. */
typedef struct EthLocator
{
  struct rte_ether_addr local;
  struct rte_ether_addr remote;
  uint16_t vlan;

  uint8_t localIP[16];
  uint8_t remoteIP[16];
  uint16_t localUDP;
  uint16_t remoteUDP;

  uint32_t vxlan;
  struct rte_ether_addr innerLocal;
  struct rte_ether_addr innerRemote;
} EthLocator;

/** @brief Determine whether two locators can coexist on the same port. */
__attribute__((nonnull)) bool
EthLocator_CanCoexist(const EthLocator* a, const EthLocator* b);

typedef struct EthRxMatch EthRxMatch;

typedef bool (*EthRxMatchFunc)(const EthRxMatch* match, const struct rte_mbuf* m);

struct EthRxMatch
{
  EthRxMatchFunc f;
  uint8_t len;
  uint8_t l2len;
  uint8_t l2matchLen;
  uint8_t l3matchOff;
  uint8_t l3matchLen;
  uint8_t udpOff;
  uint8_t buf[ETHHDR_MAXLEN];
};

/** @brief Prepare RX matcher from locator. */
__attribute__((nonnull)) void
EthRxMatch_Prepare(EthRxMatch* match, const EthLocator* loc);

/**
 * @brief Determine whether a received frame matches the locator.
 * @param match EthRxMatch prepared by @c EthRxMatch_Prepare .
 * @post if matching, the header is stripped.
 */
__attribute__((nonnull)) static inline bool
EthRxMatch_Match(const EthRxMatch* match, struct rte_mbuf* m)
{
  if (m->data_len >= match->len && (match->f)(match, m)) {
    rte_pktmbuf_adj(m, match->len);
    return true;
  }
  return false;
}

typedef struct EthFlowPattern
{
  struct rte_flow_item pattern[6];
  struct rte_flow_item_eth ethSpec;
  struct rte_flow_item_eth ethMask;
  struct rte_flow_item_vlan vlanSpec;
  struct rte_flow_item_vlan vlanMask;
  struct rte_flow_item_ipv4 ip4Spec;
  struct rte_flow_item_ipv4 ip4Mask;
  struct rte_flow_item_ipv6 ip6Spec;
  struct rte_flow_item_ipv6 ip6Mask;
  struct rte_flow_item_udp udpSpec;
  struct rte_flow_item_udp udpMask;
  struct rte_flow_item_vxlan vxlanSpec;
  struct rte_flow_item_vxlan vxlanMask;
} EthFlowPattern;

/** @brief Prepare rte_flow pattern from locator. */
__attribute__((nonnull)) void
EthFlowPattern_Prepare(EthFlowPattern* flow, const EthLocator* loc);

typedef struct EthTxHdr EthTxHdr;

typedef void (*EthTxHdrFunc)(const EthTxHdr* hdr, struct rte_mbuf* m);

struct EthTxHdr
{
  EthTxHdrFunc f;
  uint8_t len;
  uint8_t l2len;
  uint8_t buf[ETHHDR_MAXLEN];
};

/** @brief Prepare TX header from locator. */
__attribute__((nonnull)) void
EthTxHdr_Prepare(EthTxHdr* hdr, const EthLocator* loc, bool hasChecksumOffloads);

/**
 * @brief Prepend TX header.
 * @param hdr prepared by @c EthTxHdr_Prepare .
 */
__attribute__((nonnull)) static inline void
EthTxHdr_Prepend(const EthTxHdr* hdr, struct rte_mbuf* m)
{
  (hdr->f)(hdr, m);
}

#endif // NDNDPDK_ETHFACE_LOCATOR_H
