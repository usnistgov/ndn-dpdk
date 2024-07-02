#ifndef NDNDPDK_ETHFACE_RXTABLE_H
#define NDNDPDK_ETHFACE_RXTABLE_H

/** @file */

#include "../iface/rxloop.h"
#include "../pdump/source.h"
#include <urcu/rculist.h>

/** @brief Table-based software RX dispatching. */
typedef struct EthRxTable {
  RxGroup base;
  struct cds_list_head head;
  struct rte_mempool* copyTo;
  uint16_t port;
  uint16_t queue;
} EthRxTable;

__attribute__((nonnull)) void
EthRxTable_Init(EthRxTable* rxt, uint16_t port);

__attribute__((nonnull)) void
EthRxTable_RxBurst(RxGroup* rxg, RxGroupBurstCtx* ctx);

#endif // NDNDPDK_ETHFACE_RXTABLE_H
