#include "pktqueue.h"

__attribute__((nonnull)) static inline PktQueuePopResult
PktQueue_PopFromRing(PktQueue* q, struct rte_mbuf* pkts[], uint32_t count)
{
  count = RTE_MIN(count, q->dequeueBurstSize);
  PktQueuePopResult res = {
    .count = rte_ring_dequeue_burst(q->ring, (void**)pkts, count, NULL),
  };
  return res;
}

PktQueuePopResult
PktQueue_PopPlain(PktQueue* q, struct rte_mbuf* pkts[], uint32_t count, TscTime now)
{
  return PktQueue_PopFromRing(q, pkts, count);
}

PktQueuePopResult
PktQueue_PopDelay(PktQueue* q, struct rte_mbuf* pkts[], uint32_t count, TscTime now)
{
  PktQueuePopResult res = PktQueue_PopFromRing(q, pkts, count);
  ;
  if (unlikely(res.count == 0)) {
    return res;
  }

  TscTime delayUntil = Mbuf_GetTimestamp(pkts[res.count - 1]) + q->target;
  if (unlikely(now < delayUntil)) {
    while (rte_get_tsc_cycles() < delayUntil) {
      rte_pause();
    }
  }
  return res;
}

#define REC_INV_SQRT_BITS (8 * sizeof(uint16_t))
#define REC_INV_SQRT_SHIFT (32 - REC_INV_SQRT_BITS)

__attribute__((nonnull)) static inline void
CoDel_NewtonStep(PktQueue* q)
{
  uint32_t invsqrt = ((uint32_t)q->recInvSqrt) << REC_INV_SQRT_BITS;
  uint32_t invsqrt2 = ((uint64_t)invsqrt * invsqrt) >> 32;
  uint64_t val = (3LL << 32) - ((uint64_t)q->count * invsqrt2);
  val >>= 2;
  val = (val * invsqrt) >> (32 - 2 + 1);
  q->recInvSqrt = val >> REC_INV_SQRT_SHIFT;
}

static inline uint32_t
CoDel_ReciprocalScale(uint32_t val, uint32_t epro)
{
  return (uint32_t)(((uint64_t)val * epro) >> 32);
}

static inline TscTime
CoDel_ControlLaw(TscTime t, TscDuration interval, uint32_t recInvSqrt)
{
  return t + CoDel_ReciprocalScale(interval, recInvSqrt << REC_INV_SQRT_SHIFT);
}

__attribute__((nonnull)) static inline bool
CoDel_ShouldDrop(PktQueue* q, TscTime timestamp, TscTime now)
{
  q->sojourn = now - timestamp;
  if (likely(q->sojourn < q->target)) {
    q->firstAboveTime = 0;
    return false;
  }
  bool drop = false;
  if (q->firstAboveTime == 0) {
    q->firstAboveTime = now + q->interval;
  } else if (now > q->firstAboveTime) {
    drop = true;
  }
  return drop;
}

PktQueuePopResult
PktQueue_PopCoDel(PktQueue* q, struct rte_mbuf* pkts[], uint32_t count, TscTime now)
{
  PktQueuePopResult res = PktQueue_PopFromRing(q, pkts, count);
  if (unlikely(res.count == 0)) {
    q->firstAboveTime = 0;
    return res;
  }
  res.drop = CoDel_ShouldDrop(q, Mbuf_GetTimestamp(pkts[0]), now);
  q->nDrops += (int)res.drop;

  if (q->dropping) {
    if (!res.drop) {
      q->dropping = false;
    } else if (now >= q->dropNext) {
      ++q->count;
      CoDel_NewtonStep(q);
      q->dropNext = CoDel_ControlLaw(q->dropNext, q->interval, q->recInvSqrt);
    }
  } else if (res.drop) {
    q->dropping = true;
    uint32_t delta = q->count - q->lastCount;
    if (delta > 1 && (TscDuration)(now - q->dropNext) < 16 * q->interval) {
      q->count = delta;
      CoDel_NewtonStep(q);
    } else {
      q->count = 1;
      q->recInvSqrt = ~0U >> REC_INV_SQRT_SHIFT;
    }
    q->lastCount = q->count;
    q->dropNext = CoDel_ControlLaw(now, q->interval, q->recInvSqrt);
  }
  return res;
}
