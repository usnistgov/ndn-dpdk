#ifndef NDNDPDK_DPDK_ETHDEV_H
#define NDNDPDK_DPDK_ETHDEV_H

/** @file */

#include "../core/common.h"
#include <rte_ethdev.h>
#include <rte_flow.h>

/** @brief Retrieve whether an Ethernet device is DOWN. */
static inline bool
EthDev_IsDown(uint16_t port) {
  struct rte_eth_link link;
  rte_eth_link_get_nowait(port, &link);
  return link.link_status == RTE_ETH_LINK_DOWN;
}

#endif // NDNDPDK_DPDK_ETHDEV_H
