#ifndef NDN_DPDK_IFACE_ETHFACE_ETH_RX_H
#define NDN_DPDK_IFACE_ETHFACE_ETH_RX_H

#include "common.h"

/// \file

typedef struct EthFace EthFace;

uint16_t EthRx_RxBurst(EthFace* face, uint16_t queue, struct rte_mbuf** pkts,
                       uint16_t nPkts);

#endif // NDN_DPDK_IFACE_ETHFACE_ETH_RX_H
