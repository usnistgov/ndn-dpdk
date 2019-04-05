#ifndef NDN_DPDK_IFACE_ETHFACE_RXLOOP_H
#define NDN_DPDK_IFACE_ETHFACE_RXLOOP_H

/// \file

#include "../rxloop.h"
#include <rte_ether.h>
#include <rte_flow.h>

/** \brief Table-based software RX dispatching.
 */
typedef struct EthRxTable
{
  RxGroup base;
  uint16_t port;
  uint16_t queue;
  FaceId multicast;    ///< multicast face
  FaceId unicast[256]; ///< unicast faces, by last octet of sender address
} EthRxTable;

uint16_t
EthRxTable_RxBurst(RxGroup* rxg, struct rte_mbuf** pkts, uint16_t nPkts);

/** \brief rte_flow-based hardware RX dispatching.
 */
typedef struct EthRxFlow
{
  RxGroup base;
  uint16_t port;
  uint16_t queue;
  FaceId face;
  struct rte_flow* flow;
} EthRxFlow;

/** \brief Setup rte_flow on EthDev for hardware dispatching.
 *  \param sender remote unicast MAC address, or NULL for multicast
 */
bool
EthRxFlow_Setup(EthRxFlow* rxf,
                struct ether_addr* sender,
                struct rte_flow_error* error);

uint16_t
EthRxFlow_RxBurst(RxGroup* rxg, struct rte_mbuf** pkts, uint16_t nPkts);

#endif // NDN_DPDK_IFACE_ETHFACE_RXLOOP_H
