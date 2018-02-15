#ifndef NDN_DPDK_DPDK_MBUF_H
#define NDN_DPDK_DPDK_MBUF_H

/// \file

#include "../core/common.h"
#include <rte_mbuf.h>

/** \brief Get private header after struct rte_mbuf.
 *  \param m pointer to struct rte_mbuf
 *  \param T type to cast result to
 *  \param off offset in private header
 */
#define MbufPriv(m, T, off) ((T)((char*)(m) + sizeof(struct rte_mbuf) + (off)))

/** \brief Get direct mbuf's private header after struct rte_mbuf.
 *  \param mi pointer to (possibly indirect) struct rte_mbuf
 *  \param T type to cast result to
 *  \param off offset in private header
 */
#define MbufDirectPriv(mi, T, off) MbufPriv(rte_mbuf_from_indirect(mi), T, off)

/** \brief Free an array of mbufs[0..count-1].
 */
static void
FreeMbufs(struct rte_mbuf* mbufs[], int count)
{
  for (int i = 0; i < count; ++i) {
    rte_pktmbuf_free(mbufs[i]);
  }
}

/** \brief Remove \p len bytes at the beginning of a packet.
 *
 *  This function does not require first segment to have enough length.
 */
static bool
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

#endif // NDN_DPDK_DPDK_MBUF_H