#ifndef NDNDPDK_DPDK_THREAD_H
#define NDNDPDK_DPDK_THREAD_H

/** @file */

#include "../core/common.h"

/**
 * @brief Flag for instructing a thread to stop.
 *
 * This should be embedded in the thread structure.
 */
typedef atomic_bool ThreadStopFlag;

static inline void
ThreadStopFlag_Init(ThreadStopFlag* flag)
{
  atomic_init(flag, true);
}

/**
 * @brief Determine if a stop has been requested.
 * @retval false the thread should stop.
 * @retval true the thread should continue execution.
 */
static inline bool
ThreadStopFlag_ShouldContinue(ThreadStopFlag* flag)
{
#ifdef NDNDPDK_THREADSLEEP
  struct timespec req = { .tv_sec = 0, .tv_nsec = 1 };
  nanosleep(&req, NULL);
#endif // NDNDPDK_THREADSLEEP
  return atomic_load_explicit(flag, memory_order_acquire);
}

static inline void
ThreadStopFlag_RequestStop(ThreadStopFlag* flag)
{
  atomic_store_explicit(flag, false, memory_order_release);
}

static inline void
ThreadStopFlag_FinishStop(ThreadStopFlag* flag)
{
  atomic_store_explicit(flag, true, memory_order_release);
}

/**
 * @brief Load statistics of a polling thread.
 *
 * This should be embedded in the thread structure.
 */
typedef struct ThreadLoadStat
{
  uint64_t nPolls[2]; // [0] empty polls; [1] valid polls
} ThreadLoadStat;

/** @brief Report number of processed packets/items after each poll. */
static __rte_always_inline void
ThreadLoadStat_Report(ThreadLoadStat* s, uint64_t count)
{
  ++s->nPolls[(int)(count > 0)];
}

#endif // NDNDPDK_DPDK_THREAD_H
