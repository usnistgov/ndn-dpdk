#ifndef NDNDPDK_SOCKETFACE_RXCONNS_H
#define NDNDPDK_SOCKETFACE_RXCONNS_H

/** @file */

#include "../iface/rxloop.h"

/** @brief RX from Go net.Conn. */
typedef struct SocketRxConns
{
  RxGroup base;
  struct rte_ring* ring;
} SocketRxConns;

__attribute__((nonnull)) void
SocketRxConns_RxBurst(RxGroup* rxg, RxGroupBurstCtx* ctx);

#endif // NDNDPDK_SOCKETFACE_RXCONNS_H
