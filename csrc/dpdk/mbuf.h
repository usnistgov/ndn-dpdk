#ifndef NDNDPDK_DPDK_MBUF_H
#define NDNDPDK_DPDK_MBUF_H

/** @file */

#include "tsc.h"
#include <rte_mbuf.h>
#include <rte_mbuf_dyn.h>
#include <rte_ring.h>

extern int Mbuf_Timestamp_DynFieldOffset_;

/** @brief Register mbuf dynfields. */
bool
Mbuf_RegisterDynFields();

/** @brief Retrieve mbuf timestamp. */
__attribute__((nonnull)) static inline TscTime
Mbuf_GetTimestamp(struct rte_mbuf* m)
{
  return *RTE_MBUF_DYNFIELD(m, Mbuf_Timestamp_DynFieldOffset_, TscTime*);
}

/** @brief Assign mbuf timestamp. */
__attribute__((nonnull)) static inline void
Mbuf_SetTimestamp(struct rte_mbuf* m, TscTime timestamp)
{
  *RTE_MBUF_DYNFIELD(m, Mbuf_Timestamp_DynFieldOffset_, TscTime*) = timestamp;
}

/**
 * @brief Copy @c m[off:off+len] into @p dst .
 * @param dst must have @p len room.
 */
__attribute__((nonnull)) static inline void
Mbuf_ReadTo(struct rte_mbuf* m, uint32_t off, uint32_t len, void* dst)
{
  const uint8_t* readTo = rte_pktmbuf_read(m, off, len, dst);
  if (readTo != dst) {
    rte_memcpy(dst, readTo, len);
  }
}

/**
 * @brief Gather segments of @c m as iovec.
 * @param[out] iov must have @c m->nb_segs room.
 * @return iov count.
 */
__attribute__((nonnull)) int
Mbuf_AsIovec(struct rte_mbuf* m, struct iovec* iov, uint32_t offset, uint32_t length);

/**
 * @brief Allocate room within chained mbufs.
 * @param mp mempool to create mbufs.
 * @param[out] iov room iov, must be filled/zeroed by caller.
 * @param[inout] iovcnt @p iov capacity; room iov count.
 * @param firstHeadroom headroom in first mbuf.
 * @param firstDataLen data length in first mbuf.
 * @param eachHeadroom headroom in subsequent mbuf.
 * @param eachDataLen data length in subsequent mbuf.
 * @param pktLen total packet length i.e. size of room.
 * @return chained mbuf, or NULL upon failure.
 * @post @c rte_errno=E2BIG if either @c firstHeadroom+firstDataLen or
 *       @c eachHeadroom+eachDataLen exceeds @p mp dataroom.
 * @post @c rte_errno=EFBIG if number of segments would exceed @c *iovcnt .
 */
__attribute__((nonnull)) struct rte_mbuf*
Mbuf_AllocRoom(struct rte_mempool* mp, struct iovec* iov, int* iovcnt, uint16_t firstHeadroom,
               uint16_t firstDataLen, uint16_t eachHeadroom, uint16_t eachDataLen, uint32_t pktLen);

/**
 * @brief Recover remaining iovecs after @c spdk_iov_xfer_* operations.
 * @param ix initialized io_xfer instance.
 * @param[out] iov remaining iov, must have capacity as @c spdk_iov_xfer_init iovcnt.
 * @param[out] iovcnt remaining iov count.
 */
__attribute__((nonnull)) void
Mbuf_RemainingIovec(struct spdk_iov_xfer ix, struct iovec* iov, int* iovcnt);

/**
 * @brief Chain @p tail onto @p head.
 * @param lastSeg must be rte_pktmbuf_lastseg(head)
 * @return whether success.
 */
__attribute__((nonnull, warn_unused_result)) static inline bool
Mbuf_Chain(struct rte_mbuf* head, struct rte_mbuf* lastSeg, struct rte_mbuf* tail)
{
  NDNDPDK_ASSERT(lastSeg == rte_pktmbuf_lastseg(head));

  if (unlikely(head->nb_segs + tail->nb_segs > RTE_MBUF_MAX_NB_SEGS)) {
    return false;
  }

  lastSeg->next = tail;
  head->nb_segs += tail->nb_segs;
  head->pkt_len += tail->pkt_len;
  return true;
}

/**
 * @brief Chain a vector of mbufs together.
 * @param vec a non-empty vector of mbufs, each must be unsegmented.
 */
__attribute__((nonnull, returns_nonnull)) static inline struct rte_mbuf*
Mbuf_ChainVector(struct rte_mbuf* vec[], uint16_t count)
{
  NDNDPDK_ASSERT(count > 0);
  static_assert(UINT16_MAX <= RTE_MBUF_MAX_NB_SEGS, ""); // count <= RTE_MBUF_MAX_NB_SEGS
  struct rte_mbuf* head = vec[0];
  NDNDPDK_ASSERT(rte_pktmbuf_is_contiguous(head));

  for (uint16_t i = 1; i < count; ++i) {
    NDNDPDK_ASSERT(rte_pktmbuf_is_contiguous(vec[i]));
    head->pkt_len += vec[i]->data_len;
    vec[i - 1]->next = vec[i];
  }
  head->nb_segs = count;
  return head;
}

/**
 * @brief Enqueue a burst of packets to a ring buffer.
 * @param autoFree if true, rejected packets are freed.
 * @return number of rejected packets.
 */
__attribute__((nonnull)) static __rte_always_inline uint32_t
Mbuf_EnqueueVector(struct rte_mbuf* vec[], uint32_t count, struct rte_ring* ring, bool autoFree)
{
  uint32_t nEnq = rte_ring_enqueue_burst(ring, (void**)vec, count, NULL);
  uint32_t nRej = count - nEnq;
  if (autoFree && unlikely(nRej > 0)) {
    rte_pktmbuf_free_bulk(&vec[nEnq], nRej);
  }
  return nRej;
}

#endif // NDNDPDK_DPDK_MBUF_H
