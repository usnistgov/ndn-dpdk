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
} __rte_packed PktQueuePopAct;

/**
 * @brief Thread-safe packet queue.
 *
 * It can operate in one of these modes:
 * @li plain mode: packets are dequeued as fast as possible.
 * @li delay mode: packets are dequeued no earlier than @c q->target after it's received.
 * @li CoDel mode: @c PktQueuePopResult.drop is set according to CoDel algorithm.
 */
typedef struct PktQueue
{
  struct rte_ring* ring;     ///< ringbuffer of packets in queue
  TscDuration target;        ///< delay target or CoDel target
  TscDuration interval;      ///< CoDel interval
  uint32_t dequeueBurstSize; ///< maximum dequeue burst size
  uint32_t count;            ///< CoDel internal variable
  uint32_t lastCount;        ///< CoDel internal variable
  uint16_t recInvSqrt;       ///< CoDel internal variable
  bool dropping;             ///< CoDel internal variable
  PktQueuePopAct pop;        ///< dequeue function index
  TscTime firstAboveTime;    ///< CoDel internal variable
  TscTime dropNext;          ///< CoDel internal variable
  TscDuration sojourn;       ///< CoDel internal variable
  uint64_t nDrops;           ///< number of packets marked as dropped by CoDel
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
