#ifndef NDNDPDK_IFACE_RXLOOP_H
#define NDNDPDK_IFACE_RXLOOP_H

/** @file */

#include "../dpdk/thread.h"
#include "face.h"
#include "input-demux.h"

typedef struct RxGroup RxGroup;

/**
 * @brief Receive a burst of L2 frames.
 * @param[out] pkts L2 frames; port and timestamp should be set.
 * @return successfully received frames.
 */
typedef uint16_t (*RxGroup_RxBurst)(RxGroup* rxg, struct rte_mbuf** pkts, uint16_t nPkts);

/** @brief Receive channel for a group of faces. */
typedef struct RxGroup
{
  struct cds_hlist_node rxlNode;
  RxGroup_RxBurst rxBurstOp;
  int rxThread; ///< RX thread number for RxProc_Input
} RxGroup;

extern RxGroup theChanRxGroup_;

/** @brief RX loop thread. */
typedef struct RxLoop
{
  ThreadCtrl ctrl;
  InputDemux demuxI;
  InputDemux demuxD;
  InputDemux demuxN;

  struct cds_hlist_head head;
} RxLoop;

__attribute__((nonnull)) int
RxLoop_Run(RxLoop* rxl);

#endif // NDNDPDK_IFACE_RXLOOP_H
