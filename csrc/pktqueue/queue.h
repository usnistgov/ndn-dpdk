#ifndef NDN_DPDK_PKTQUEUE_QUEUE_H
#define NDN_DPDK_PKTQUEUE_QUEUE_H

/// \file

#include "../dpdk/mbuf.h"
#include "../dpdk/tsc.h"
#include <rte_ring.h>

#define PKTQUEUE_BURST_SIZE_MAX 64

/** \brief A packet queue with simplified CoDel algorithm.
 */
typedef struct PktQueue PktQueue;

typedef struct PktQueuePopResult
{
  uint32_t count; ///< number of dequeued packets
  bool drop;      ///< whether the first packet should be dropped/ECN-marked
} PktQueuePopResult;

typedef PktQueuePopResult (*PktQueue_PopOp)(PktQueue* q, struct rte_mbuf* pkts[], uint32_t count,
                                            TscTime now);

struct PktQueue
{
  struct rte_ring* ring;

  PktQueue_PopOp pop;
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
};

/** \brief Enqueue a burst of packets.
 *  \param pkts packets with timestamp already set.
 *  \return number of rejected packets; they have been freed.
 */
static inline uint32_t
PktQueue_PushPlain(PktQueue* q, struct rte_mbuf* pkts[], uint32_t count)
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
PktQueue_Push(PktQueue* q, struct rte_mbuf* pkts[], uint32_t count, TscTime now)
{
  for (uint32_t i = 0; i < count; ++i) {
    pkts[i]->timestamp = now;
  }
  return PktQueue_PushPlain(q, pkts, count);
}

/** \brief Dequeue a burst of packets.
 */
static inline PktQueuePopResult
PktQueue_Pop(PktQueue* q, struct rte_mbuf* pkts[], uint32_t count, TscTime now)
{
  return (*q->pop)(q, pkts, count, now);
}

PktQueuePopResult
PktQueue_PopPlain(PktQueue* q, struct rte_mbuf* pkts[], uint32_t count, TscTime now);

PktQueuePopResult
PktQueue_PopDelay(PktQueue* q, struct rte_mbuf* pkts[], uint32_t count, TscTime now);

PktQueuePopResult
PktQueue_PopCoDel(PktQueue* q, struct rte_mbuf* pkts[], uint32_t count, TscTime now);

#endif // NDN_DPDK_PKTQUEUE_QUEUE_H
