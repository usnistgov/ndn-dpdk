#ifndef NDN_DPDK_CONTAINER_CODEL_QUEUE_QUEUE_H
#define NDN_DPDK_CONTAINER_CODEL_QUEUE_QUEUE_H

/// \file

#include "../../dpdk/mbuf.h"
#include "../../dpdk/tsc.h"
#include <rte_ring.h>

#define CODELQUEUE_BURST_SIZE_MAX 64

/** \brief A packet queue with simplified CoDel algorithm.
 */
typedef struct CoDelQueue
{
  struct rte_ring* ring;

  TscDuration target;
  TscDuration interval;
  uint32_t dequeueBurstSize;

  uint32_t count;
  uint32_t lastCount;
  bool dropping;
  uint16_t recInvSqrt;
  TscTime firstAboveTime;
  TscTime dropNext;
  TscDuration sojourn;

  uint64_t nDrops;
} CoDelQueue;

/** \brief Enqueue a burst of packets.
 *  \param pkts packets with timestamp already set.
 *  \return number of rejected packets; they have been freed.
 */
static inline uint32_t
CoDelQueue_PushPlain(CoDelQueue* q, struct rte_mbuf* pkts[], uint32_t count)
{
  uint32_t nEnq = rte_ring_enqueue_burst(q->ring, (void**)pkts, count, NULL);
  uint32_t nRej = count - nEnq;
  if (unlikely(nRej > 0)) {
    FreeMbufs(&pkts[nEnq], nRej);
  }
  return nRej;
}

/** \brief Set timestamp on a burst of packets and enqueue them.
 *  \return number of rejected packets; they have been freed.
 */
static inline uint32_t
CoDelQueue_Push(CoDelQueue* q,
                struct rte_mbuf* pkts[],
                uint32_t count,
                TscTime now)
{
  for (uint32_t i = 0; i < count; ++i) {
    pkts[i]->timestamp = now;
  }
  return CoDelQueue_PushPlain(q, pkts, count);
}

typedef struct CoDelPopResult
{
  uint32_t count; ///< number of dequeued packets
  bool drop;      ///< whether CoDel wants to drop/mark one packet
} CoDelPopResult;

/** \brief Dequeue a burst of packets.
 */
CoDelPopResult
CoDelQueue_Pop(CoDelQueue* q,
               struct rte_mbuf* pkts[],
               uint32_t count,
               TscTime now);

#endif // NDN_DPDK_CONTAINER_CODEL_QUEUE_QUEUE_H
