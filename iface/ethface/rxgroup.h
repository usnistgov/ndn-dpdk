#ifndef NDN_DPDK_IFACE_ETHFACE_RXLOOP_H
#define NDN_DPDK_IFACE_ETHFACE_RXLOOP_H

/// \file

#include "../rxloop.h"
#include "eth-face.h"

typedef struct EthRxGroup
{
  RxGroup base;
  uint16_t port;
  uint16_t queue;
  FaceId multicast;
  FaceId unicast[256];
} EthRxGroup;

uint16_t EthRxGroup_RxBurst(RxGroup* rxg0, struct rte_mbuf** pkts,
                            uint16_t nPkts);

#endif // NDN_DPDK_IFACE_ETHFACE_RXLOOP_H
