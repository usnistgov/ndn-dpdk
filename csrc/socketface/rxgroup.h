#ifndef NDNDPDK_SOCKETFACE_RXGROUP_H
#define NDNDPDK_SOCKETFACE_RXGROUP_H

/** @file */

#include "../iface/rxloop.h"

/** @brief Table-based software RX dispatching. */
typedef struct SocketRxGroup
{
  RxGroup base;
  struct rte_ring* ring;
} SocketRxGroup;

__attribute__((nonnull)) uint16_t
SocketRxGroup_RxBurst(RxGroup* rxg, struct rte_mbuf** pkts, uint16_t nPkts);

#endif // NDNDPDK_SOCKETFACE_RXGROUP_H
