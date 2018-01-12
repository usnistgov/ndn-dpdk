#ifndef NDN_DPDK_DPDK_ETHDEV_H
#define NDN_DPDK_DPDK_ETHDEV_H

/// \file

#include "../core/common.h"
#include <rte_ethdev.h>

/** \brief Retrieve hardware address of Ethernet device.
 *
 *  DPDK's pcap PMD returns the same MAC address on every node.
 *  This function returns a random MAC address if detecting pcap PMD's default MAC address.
 *  However, the returned MAC address would be different upon every invocation.
 *
 *  In all other cases, this function is equivalent to \p rte_eth_macaddr_get.
 */
void EthDev_GetMacAddr(uint16_t port, struct ether_addr* macaddr);

#endif // NDN_DPDK_DPDK_ETHDEV_H