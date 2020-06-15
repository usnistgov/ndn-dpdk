#ifndef NDN_DPDK_DPDK_MBUF_H
#define NDN_DPDK_DPDK_MBUF_H

/// \file

#include "../core/common.h"
#include <rte_mbuf.h>

/** \brief Free an array of mbufs[0..count-1].
 */
static __rte_always_inline uint32_t
FreeMbufs(struct rte_mbuf* mbufs[], int count)
{
  uint32_t totalLen = 0;
  for (int i = 0; i < count; ++i) {
    totalLen += mbufs[i]->pkt_len;
    rte_pktmbuf_free(mbufs[i]);
  }
  return totalLen;
}

/** \brief Remove \p len bytes at the beginning of a packet.
 *
 *  This function does not require first segment to have enough length.
 */
static inline bool
Packet_Adj(struct rte_mbuf* pkt, uint16_t len)
{
  if (unlikely(pkt->pkt_len < len)) {
    return false;
  }

  if (likely(pkt->data_len >= len)) {
    rte_pktmbuf_adj(pkt, len);
    return true;
  }

  pkt->pkt_len -= len;

  struct rte_mbuf* segment = pkt;
  while (segment != NULL && segment->data_len < len) {
    len -= segment->data_len;
    segment->data_off += segment->data_len;
    segment->data_len = 0;
    struct rte_mbuf* next = segment->next;
    if (segment != pkt) {
      rte_pktmbuf_free(segment);
    }
    segment = next;
  }

  segment->data_off += len;
  segment->data_len -= len;
  return true;
}

/** \brief Chain \p tail onto \p head.
 *  \param lastSeg must be rte_pktmbuf_lastseg(head)
 *  \retval 0 success
 *  \retval -EOVERFLOW total segment count exceeds limit
 */
static inline int
Packet_Chain(struct rte_mbuf* head,
             struct rte_mbuf* lastSeg,
             struct rte_mbuf* tail)
{
  assert(lastSeg == rte_pktmbuf_lastseg(head));

  if (unlikely(head->nb_segs + tail->nb_segs > RTE_MBUF_MAX_NB_SEGS)) {
    return -EOVERFLOW;
  }

  lastSeg->next = tail;
  head->nb_segs += tail->nb_segs;
  head->pkt_len += tail->pkt_len;
  return 0;
}

static __rte_always_inline void*
rte_pktmbuf_mtod_offset_(struct rte_mbuf* m, uint16_t offset)
{
  // turn macro into function to be called in Go
  return rte_pktmbuf_mtod_offset(m, void*, offset);
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
