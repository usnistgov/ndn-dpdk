#ifndef NDNDPDK_DPDK_MBUF_H
#define NDNDPDK_DPDK_MBUF_H

/** @file */

#include "../core/common.h"
#include <rte_mbuf.h>

/**
 * @brief Copy contents of mbuf to a buffer.
 * @param[out] dst destination buffer, must have sufficient size.
 */
__attribute__((nonnull)) static inline void
Mbuf_CopyTo(struct rte_mbuf* m, void* dst)
{
  for (struct rte_mbuf* s = m; s != NULL; s = s->next) {
    rte_memcpy(dst, rte_pktmbuf_mtod(s, const void*), s->data_len);
    dst = RTE_PTR_ADD(dst, s->data_len);
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

#endif // NDNDPDK_DPDK_MBUF_H
