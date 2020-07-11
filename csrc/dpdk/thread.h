#ifndef NDN_DPDK_DPDK_THREAD_H
#define NDN_DPDK_DPDK_THREAD_H

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

#endif // NDN_DPDK_DPDK_THREAD_H
