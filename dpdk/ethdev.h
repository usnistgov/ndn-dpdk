#ifndef NDN_DPDK_DPDK_ETHDEV_H
#define NDN_DPDK_DPDK_ETHDEV_H

/// \file

#include "../core/common.h"
#include <rte_ethdev.h>

/** \brief Retrieve whether an Ethernet device is DOWN.
 */
static bool
EthDev_IsDown(uint16_t port)
{
  struct rte_eth_link link;
  rte_eth_link_get_nowait(port, &link);
  return link.link_status == ETH_LINK_DOWN;
}

#endif // NDN_DPDK_DPDK_ETHDEV_H
