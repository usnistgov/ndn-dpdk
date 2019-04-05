#ifndef NDN_DPDK_IFACE_TXLOOP_H
#define NDN_DPDK_IFACE_TXLOOP_H

/// \file

#include "../dpdk/thread.h"

#include "face.h"

/** \brief TX loop for faces that enabled thread-safe TX.
 */
typedef struct TxLoop
{
  struct cds_hlist_head head;
  ThreadStopFlag stop;
} TxLoop;

void
TxLoop_Run(TxLoop* txl);

#endif // NDN_DPDK_IFACE_TXLOOP_H
