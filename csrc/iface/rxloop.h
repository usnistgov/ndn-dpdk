#ifndef NDN_DPDK_IFACE_RXLOOP_H
#define NDN_DPDK_IFACE_RXLOOP_H

/// \file

#include "../dpdk/thread.h"
#include "face.h"

typedef struct RxGroup RxGroup;

/** \brief Receive a burst of L2 frames.
 *  \param[out] pkts L2 frames; port and timestamp should be set.
 *  \return successfully received frames.
 */
typedef uint16_t (*RxGroup_RxBurst)(RxGroup* rxg, struct rte_mbuf** pkts, uint16_t nPkts);

/** \brief Receive channel for a group of faces.
 */
typedef struct RxGroup
{
  struct cds_hlist_node rxlNode;
  RxGroup_RxBurst rxBurstOp;
  int rxThread; ///< RX thread number for FaceImpl_RxBurst
} RxGroup;

extern RxGroup theChanRxGroup_;

/** \brief RX loop.
 */
typedef struct RxLoop
{
  FaceRxBurst* burst;
  Face_RxCb cb;
  void* cbarg;

  struct cds_hlist_head head;
  ThreadStopFlag stop;
} RxLoop;

void
RxLoop_Run(RxLoop* rxl);

#endif // NDN_DPDK_IFACE_RXLOOP_H
