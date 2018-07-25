#ifndef NDN_DPDK_IFACE_ETHFACE_RXLOOP_H
#define NDN_DPDK_IFACE_ETHFACE_RXLOOP_H

/// \file

#include "eth-face.h"

typedef struct EthRxTask
{
  uint16_t port;
  uint16_t queue;
  int rxThread;
  Face* face;
} EthRxTask;
static_assert(sizeof(EthRxTask) == 16, "");

/** \brief Ethernet RX loop.
 */
typedef struct EthRxLoop
{
  int nTasks;
  int maxTasks;
  bool stop; ///< true tells EthRxLoop_Run to stop

  /** \brief RX callbacks.
   *
   *  This array is separate from EthRxTask because this is rarely used.
   */
  const struct rte_eth_rxtx_callback** callback;

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

int EthRxLoop_AddTask(EthRxLoop* rxl, EthRxTask* task);

void EthRxLoop_Run(EthRxLoop* rxl, FaceRxBurst* burst, Face_RxCb cb,
                   void* cbarg);

#endif // NDN_DPDK_IFACE_ETHFACE_RXLOOP_H
