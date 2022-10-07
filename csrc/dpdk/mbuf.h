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
__attribute__((nonnull)) static inline void
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
