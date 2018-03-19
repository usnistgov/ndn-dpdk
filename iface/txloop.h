#ifndef NDN_DPDK_IFACE_TXLOOP_H
#define NDN_DPDK_IFACE_TXLOOP_H

/// \file

#include "face.h"

/** \brief TX loop for faces that enabled thread-safe TX.
 *  \note Currently this only supports one face.
 */
typedef struct FaceTxLoop
{
  Face* head;
  bool stop;
} FaceTxLoop;

void FaceTxLoop_Run(FaceTxLoop* txl);

#endif // NDN_DPDK_IFACE_TXLOOP_H
