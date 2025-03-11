#ifndef NDNDPDK_DPDK_MBUF_H
#define NDNDPDK_DPDK_MBUF_H

/** @file */

#include "tsc.h"
#include <rte_mbuf.h>
#include <rte_mbuf_dyn.h>
#include <rte_ring.h>

enum {
  /** @brief @c mbuf.ol_flags bits to indicate MARK action was applied. */
  Mbuf_HasMark = RTE_MBUF_F_RX_FDIR | RTE_MBUF_F_RX_FDIR_ID,
};

extern int Mbuf_Timestamp_DynFieldOffset_;

/** @brief Register mbuf dynfields. */
bool
Mbuf_RegisterDynFields();

/** @brief Retrieve mbuf timestamp. */
__attribute__((nonnull)) static inline TscTime
Mbuf_GetTimestamp(struct rte_mbuf* m) {
  return *RTE_MBUF_DYNFIELD(m, Mbuf_Timestamp_DynFieldOffset_, TscTime*);
}

/** @brief Assign mbuf timestamp. */
__attribute__((nonnull)) static inline void
Mbuf_SetTimestamp(struct rte_mbuf* m, TscTime timestamp) {
  *RTE_MBUF_DYNFIELD(m, Mbuf_Timestamp_DynFieldOffset_, TscTime*) = timestamp;
}

/** @brief Retrieve mbuf MARK action value. */
__attribute__((nonnull)) static inline uint32_t
Mbuf_GetMark(const struct rte_mbuf* m) {
  if ((m->ol_flags & Mbuf_HasMark) != Mbuf_HasMark) {
    return 0;
  }
  return m->hash.fdir.hi;
}

/** @brief Assign mbuf MARK action value. */
__attribute__((nonnull)) static inline void
Mbuf_SetMark(struct rte_mbuf* m, uint32_t value) {
  m->ol_flags |= Mbuf_HasMark;
  m->hash.fdir.hi = value;
}

/**
 * @brief Copy @c m[off:off+len] into @p dst .
 * @param dst must have @p len room.
 */
__attribute__((nonnull)) static inline void
Mbuf_ReadTo(const struct rte_mbuf* m, uint32_t off, uint32_t len, void* dst) {
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
 * @param firstDataLen data length in first mbuf, defaults to maximum available.
 * @param eachHeadroom headroom in subsequent mbuf.
 * @param eachDataLen data length in subsequent mbuf, defaults to maximum available.
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
 * @post in case of failure, both mbufs are freed.
 */
__attribute__((nonnull, warn_unused_result)) static inline bool
Mbuf_Chain(struct rte_mbuf* head, struct rte_mbuf* lastSeg, struct rte_mbuf* tail) {
  if (unlikely(head->nb_segs + tail->nb_segs > RTE_MBUF_MAX_NB_SEGS)) {
    struct rte_mbuf* mbufs[2] = {head, tail};
    rte_pktmbuf_free_bulk(mbufs, RTE_DIM(mbufs));
    return false;
  }

  lastSeg->next = tail;
  head->nb_segs += tail->nb_segs;
  head->pkt_len += tail->pkt_len;
  return true;
}

/**
 * @brief Chain a vector of mbufs together.
 * @param vec a vector of mbufs, may contain NULL, each mbuf is possibly segmented.
 * @return chained mbuf.
 * @retval NULL failure, or all elements in vector are NULL.
 * @post in case of failure, all mbufs are freed.
 */
__attribute__((nonnull, warn_unused_result)) static inline struct rte_mbuf*
Mbuf_ChainVector(struct rte_mbuf* vec[], uint16_t count) {
  struct rte_mbuf* head = NULL;
  struct rte_mbuf* last = NULL;
  for (uint16_t i = 0; i < count; ++i) {
    struct rte_mbuf* m = vec[i];
    if (unlikely(m == NULL)) {
      continue;
    }

    if (head == NULL) {
      head = m;
      last = rte_pktmbuf_lastseg(m);
      continue;
    }

    struct rte_mbuf* mLast = rte_pktmbuf_lastseg(m);
    bool ok = Mbuf_Chain(head, last, m);
    if (unlikely(!ok)) {
      ++i;
      rte_pktmbuf_free_bulk(&vec[i], count - i);
      return NULL;
    }
    last = mLast;
  }
  return head;
}

/**
 * @brief Free a sequence of segments.
 * @param begin pointer to first segment pointer.
 * @param end exclusive last segment, will not be freed.
 * @param pktLen pointer to packet length, will be decremented.
 * @return number of freed segments.
 * @post @c *begin is freed, but @c end is not freed.
 * @post *begin == end
 * @code
 * // free second through last segments of an mbuf
 * pkt->nb_segs -= Mbuf_FreeSegs(&pkt->next, NULL, &pkt->pkt_len);
 * @endcode
 */
__attribute__((nonnull(1, 3))) static inline uint16_t
Mbuf_FreeSegs(struct rte_mbuf** begin, struct rte_mbuf* end, uint32_t* pktLen) {
  uint16_t count = 0;
  for (struct rte_mbuf* seg = *begin; seg != end;) {
    struct rte_mbuf* next = seg->next;
    ++count;
    *pktLen -= seg->data_len;
    rte_pktmbuf_free_seg(seg);
    seg = next;
  }
  *begin = end;
  return count;
}

/**
 * @brief Enqueue a burst of packets to a ring buffer.
 * @param autoFree if true, rejected packets are freed.
 * @return number of rejected packets.
 */
__attribute__((nonnull)) static __rte_always_inline uint32_t
Mbuf_EnqueueVector(struct rte_mbuf* vec[], uint32_t count, struct rte_ring* ring, bool autoFree) {
  uint32_t nEnq = rte_ring_enqueue_burst(ring, (void**)vec, count, NULL);
  uint32_t nRej = count - nEnq;
  if (autoFree && unlikely(nRej > 0)) {
    rte_pktmbuf_free_bulk(&vec[nEnq], nRej);
  }
  return nRej;
}

#endif // NDNDPDK_DPDK_MBUF_H
