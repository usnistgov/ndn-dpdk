#ifndef NDN_DPDK_IFACE_ETHFACE_ETH_RX_H
#define NDN_DPDK_IFACE_ETHFACE_ETH_RX_H

/// \file

#include "common.h"

typedef struct EthFace EthFace;

/** \brief Ethernet receiving queue.
 */
typedef struct EthRx
{
} EthRx;

uint16_t EthRx_RxBurst(EthFace* face, uint16_t queue, struct rte_mbuf** pkts,
                       uint16_t nPkts);

#endif // NDN_DPDK_IFACE_ETHFACE_ETH_RX_H
