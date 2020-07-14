#ifndef NDNDPDK_ETHFACE_RXTABLE_H
#define NDNDPDK_ETHFACE_RXTABLE_H

/** @file */

#include "../iface/rxloop.h"

/** @brief Table-based software RX dispatching. */
typedef struct EthRxTable
{
  RxGroup base;
  uint16_t port;
  uint16_t queue;
  _Atomic FaceID multicast;    ///< multicast face
  _Atomic FaceID unicast[256]; ///< unicast faces, by last octet of sender address
} EthRxTable;

uint16_t
EthRxTable_RxBurst(RxGroup* rxg, struct rte_mbuf** pkts, uint16_t nPkts);

#endif // NDNDPDK_ETHFACE_RXTABLE_H
