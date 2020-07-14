#ifndef NDN_DPDK_DPDK_MBUF_H
#define NDN_DPDK_DPDK_MBUF_H

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
    rte_memcpy(dst, rte_pktmbuf_mtod(s, void*), s->data_len);
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
  assert(lastSeg == rte_pktmbuf_lastseg(head));

  if (unlikely(head->nb_segs + tail->nb_segs > RTE_MBUF_MAX_NB_SEGS)) {
    return false;
  }

  lastSeg->next = tail;
  head->nb_segs += tail->nb_segs;
  head->pkt_len += tail->pkt_len;
  return true;
}

static __rte_always_inline void*
rte_mbuf_to_priv_(struct rte_mbuf* m)
{
#pragma GCC diagnostic push
#pragma GCC diagnostic ignored "-Wdeprecated-declarations"
  return rte_mbuf_to_priv(m);
#pragma GCC diagnostic pop
}

static __rte_always_inline void
rte_pktmbuf_free_bulk_(struct rte_mbuf** mbufs, unsigned int count)
{
#pragma GCC diagnostic push
#pragma GCC diagnostic ignored "-Wdeprecated-declarations"
  rte_pktmbuf_free_bulk(mbufs, count);
#pragma GCC diagnostic pop
}

#endif // NDN_DPDK_DPDK_MBUF_H
