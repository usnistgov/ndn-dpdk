#ifndef NDN_DPDK_IFACE_ETHFACE_RXLOOP_H
#define NDN_DPDK_IFACE_ETHFACE_RXLOOP_H

/// \file

#include "eth-face.h"

typedef struct EthRxTask
{
  uint16_t port;
  uint16_t queue;
  int rxThread;
  FaceId multicast;
  FaceId unicast[256];
} __rte_cache_aligned EthRxTask;

/** \brief Ethernet RX loop.
 */
typedef struct EthRxLoop
{
  int nTasks;
  int maxTasks;
  bool stop; ///< true tells EthRxLoop_Run to stop

  /** \brief Array of tasks.
   */
  EthRxTask task[0];
} EthRxLoop;

EthRxLoop* EthRxLoop_New(int maxTasks, int numaSocket);

void EthRxLoop_Close(EthRxLoop* rxl);

static EthRxTask*
__EthRxLoop_GetTask(EthRxLoop* rxl, int i)
{
  return &rxl->task[i];
}

void EthRxLoop_AddTask(EthRxLoop* rxl, EthRxTask* task);

void EthRxLoop_Run(EthRxLoop* rxl, FaceRxBurst* burst, Face_RxCb cb,
                   void* cbarg);

#endif // NDN_DPDK_IFACE_ETHFACE_RXLOOP_H
