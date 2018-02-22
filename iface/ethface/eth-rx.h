#ifndef NDN_DPDK_IFACE_ETHFACE_ETH_RX_H
#define NDN_DPDK_IFACE_ETHFACE_ETH_RX_H

/// \file

#include "common.h"

typedef struct EthFace EthFace;

/** \brief Ethernet receiving queue.
 */
typedef struct EthRx
{
  void* rxCallback;
} EthRx;

/** \brief Initialize Ethernet TX
 *  \return 0 for success, otherwise error code
 */
int EthRx_Init(EthFace* face, uint16_t queue);

uint16_t EthRx_RxBurst(EthFace* face, uint16_t queue, struct rte_mbuf** pkts,
                       uint16_t nPkts);

#endif // NDN_DPDK_IFACE_ETHFACE_ETH_RX_H
