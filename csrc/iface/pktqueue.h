#ifndef NDNDPDK_IFACE_PKTQUEUE_H
#define NDNDPDK_IFACE_PKTQUEUE_H

/** @file */

#include "common.h"

/** @brief Packet queue dequeue method. */
typedef enum PktQueuePopAct
{
  PktQueuePopActPlain,
  PktQueuePopActDelay,
  PktQueuePopActCoDel,
} PktQueuePopAct;

/** @brief Thread-safe packet queue. */
typedef struct PktQueue
{
  struct rte_ring* ring;

  PktQueuePopAct pop;
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
} PktQueue;

/**
 * @brief Enqueue a burst of packets.
 * @param pkts packets with timestamp assigned.
 * @return number of rejected packets; caller must free them.
 */
__attribute__((nonnull)) static inline uint32_t
PktQueue_Push(PktQueue* q, struct rte_mbuf* pkts[], uint32_t count)
{
  return Mbuf_EnqueueVector(pkts, count, q->ring, false);
}

/** @brief Packet queue pop result. */
typedef struct PktQueuePopResult
{
  uint32_t count; ///< number of dequeued packets
  bool drop;      ///< whether the first packet should be dropped/ECN-marked
} PktQueuePopResult;

typedef PktQueuePopResult (*PktQueue_PopFunc)(PktQueue* q, struct rte_mbuf* pkts[], uint32_t count,
                                              TscTime now);
extern const PktQueue_PopFunc PktQueue_PopJmp[];

/** @brief Dequeue a burst of packets. */
__attribute__((nonnull)) static inline PktQueuePopResult
PktQueue_Pop(PktQueue* q, struct rte_mbuf* pkts[], uint32_t count, TscTime now)
{
  return PktQueue_PopJmp[q->pop](q, pkts, count, now);
}

#endif // NDNDPDK_IFACE_PKTQUEUE_H
