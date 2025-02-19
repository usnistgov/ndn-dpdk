#ifndef NDNDPDK_IFACE_RXLOOP_H
#define NDNDPDK_IFACE_RXLOOP_H

/** @file */

#include "../dpdk/thread.h"
#include "face.h"
#include "input-demux.h"
#include <rte_bitset.h>

/** @brief Context of RxGroup_RxBurstFunc operation. */
typedef struct RxGroupBurstCtx {
  RTE_BITSET_DECLARE(dropBits, MaxBurstSize);
  uint16_t nRx;
  RTE_MARKER zeroizeEnd_;
  struct rte_mbuf* pkts[MaxBurstSize];
} RxGroupBurstCtx;

/** @brief Mark @c ctx->pkts[i] as to be dropped. */
__attribute__((nonnull)) static inline void
RxGroupBurstCtx_Drop(RxGroupBurstCtx* ctx, uint16_t i) {
  rte_bitset_set(ctx->dropBits, i);
}

typedef struct RxGroup RxGroup;

/**
 * @brief Receive a burst of L2 frames.
 * @pre @c ctx->nRx and @c ctx->dropBits are zero.
 *
 * The callback should fill @c ctx->pkts[:ctx->nRx] with received packets, and set @c pkt->port
 * (FaceID) and timestamp on each packet.
 * The callback may mark an index with @c RxGroupBurstCtx_Drop so that they would be freed by the
 * caller without processing; these positions may also have NULL.
 */
typedef void (*RxGroup_RxBurstFunc)(RxGroup* rxg, RxGroupBurstCtx* ctx);

/** @brief Receive channel for faces. */
struct RxGroup {
  struct cds_hlist_node rxlNode;
  RxGroup_RxBurstFunc rxBurst;
  int rxThread; ///< FaceRx_Input thread number
};

/** @brief RX loop thread. */
typedef struct RxLoop {
  ThreadCtrl ctrl;
  InputDemuxes demuxes;

  struct cds_hlist_head head;
} RxLoop;

__attribute__((nonnull)) int
RxLoop_Run(RxLoop* rxl);

#endif // NDNDPDK_IFACE_RXLOOP_H
