#ifndef NDN_DPDK_IFACE_TXLOOP_H
#define NDN_DPDK_IFACE_TXLOOP_H

/// \file

#include "face.h"

/** \brief TX loop for multiple faces that enabled thread-safe TX.
 */
typedef struct MultiTxLoop
{
  struct cds_hlist_head head;
  bool stop;
} MultiTxLoop;

void MultiTxLoop_Run(MultiTxLoop* txl);

#endif // NDN_DPDK_IFACE_TXLOOP_H
