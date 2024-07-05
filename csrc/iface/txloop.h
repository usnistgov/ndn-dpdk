#ifndef NDNDPDK_IFACE_TXLOOP_H
#define NDNDPDK_IFACE_TXLOOP_H

/** @file */

#include "../dpdk/thread.h"
#include "face.h"

/** @brief TX loop thread. */
typedef struct TxLoop {
  ThreadCtrl ctrl;
  struct cds_hlist_head head;
} TxLoop;

/**
 * @brief Submit frames to @c face->impl->txBurst .
 * @param count must be positive.
 */
__attribute__((nonnull)) void
TxLoop_TxFrames(Face* face, int txThread, struct rte_mbuf** frames, uint16_t count);

/**
 * @brief Move a burst of L3 packets from @c face->outputQueue to @c face->impl->txBurst .
 *
 * This is the default implementation of @c Face_TxLoopFunc for @c PacketTxAlign.linearize==true .
 */
__attribute__((nonnull)) uint16_t
TxLoop_Transfer_Linear(Face* face, int txThread);

/**
 * @brief Move a burst of L3 packets from @c face->outputQueue to @c face->impl->txBurst .
 *
 * This is the default implementation of @c Face_TxLoopFunc for @c PacketTxAlign.linearize==false .
 */
__attribute__((nonnull)) uint16_t
TxLoop_Transfer_Chained(Face* face, int txThread);

__attribute__((nonnull)) int
TxLoop_Run(TxLoop* txl);

#endif // NDNDPDK_IFACE_TXLOOP_H
