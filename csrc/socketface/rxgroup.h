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

__attribute__((nonnull)) void
SocketRxGroup_RxBurst(RxGroup* rxg, RxGroupBurstCtx* ctx);

#endif // NDNDPDK_SOCKETFACE_RXGROUP_H
