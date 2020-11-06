#ifndef NDNDPDK_ETHFACE_RXTABLE_H
#define NDNDPDK_ETHFACE_RXTABLE_H

/** @file */

#include "../iface/rxloop.h"

/** @brief Table-based software RX dispatching. */
typedef struct EthRxTable
{
  RxGroup base;
  struct cds_hlist_head head;
  uint16_t port;
  uint16_t queue;
} EthRxTable;

__attribute__((nonnull)) uint16_t
EthRxTable_RxBurst(RxGroup* rxg, struct rte_mbuf** pkts, uint16_t nPkts);

#endif // NDNDPDK_ETHFACE_RXTABLE_H
