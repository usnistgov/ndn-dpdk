#ifndef NDNDPDK_IFACE_TXLOOP_H
#define NDNDPDK_IFACE_TXLOOP_H

/** @file */

#include "../dpdk/thread.h"
#include "face.h"

/** @brief TX loop thread. */
typedef struct TxLoop
{
  struct cds_hlist_head head;
  ThreadStopFlag stop;
  ThreadLoadStat loadStat;
} TxLoop;

__attribute__((nonnull)) int
TxLoop_Run(TxLoop* txl);

#endif // NDNDPDK_IFACE_TXLOOP_H
