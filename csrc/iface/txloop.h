#ifndef NDN_DPDK_IFACE_TXLOOP_H
#define NDN_DPDK_IFACE_TXLOOP_H

/// \file

#include "../dpdk/thread.h"
#include "face.h"

/** \brief TX loop.
 */
typedef struct TxLoop
{
  struct cds_hlist_head head;
  ThreadStopFlag stop;
} TxLoop;

int
TxLoop_Run(TxLoop* txl);

#endif // NDN_DPDK_IFACE_TXLOOP_H
