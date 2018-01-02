#ifndef NDN_DPDK_IFACE_ETHFACE_ETH_RX_H
#define NDN_DPDK_IFACE_ETHFACE_ETH_RX_H

#include "in-order-reassembler.h"

/// \file

typedef struct EthFace EthFace;

typedef struct EthRx
{
  uint16_t queue;

  InOrderReassembler reassembler;

  uint64_t nFrames;       ///< number of L2 frames
  uint64_t nInterestPkts; ///< number of Interests decoded
  uint64_t nDataPkts;     ///< number of Data decoded
} __rte_cache_aligned EthRx;

uint16_t EthRx_RxBurst(EthFace* face, EthRx* rx, struct rte_mbuf** pkts,
                       uint16_t nPkts);

void EthRx_ReadCounters(EthFace* face, EthRx* rx, FaceCounters* cnt);

#endif // NDN_DPDK_IFACE_ETHFACE_ETH_RX_H
