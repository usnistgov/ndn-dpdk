#ifndef NDNDPDK_IFACE_TXLOOP_H
#define NDNDPDK_IFACE_TXLOOP_H

/** @file */

#include "../dpdk/thread.h"
#include "face.h"

/** @brief TX loop thread. */
typedef struct TxLoop
{
  ThreadCtrl ctrl;
  struct cds_hlist_head head;
} TxLoop;

__attribute__((nonnull)) int
TxLoop_Run(TxLoop* txl);

#endif // NDNDPDK_IFACE_TXLOOP_H
