#ifndef NDNDPDK_DPDK_THREAD_H
#define NDNDPDK_DPDK_THREAD_H

/** @file */

#include "../core/common.h"

typedef atomic_bool ThreadStopFlag;

static inline void
ThreadStopFlag_Init(ThreadStopFlag* flag)
{
  atomic_init(flag, true);
}

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

#endif // NDNDPDK_DPDK_THREAD_H
