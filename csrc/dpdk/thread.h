#ifndef NDNDPDK_DPDK_THREAD_H
#define NDNDPDK_DPDK_THREAD_H

/** @file */

#include "../core/common.h"
#include "thread-enum.h"

/** @brief Thread load stats and stop flag. */
typedef struct ThreadCtrl
{
  uint64_t nPolls[2]; // [0] empty polls; [1] valid polls
  uint64_t items;
  uint32_t sleepFor;
  atomic_bool stop;
} ThreadCtrl;

static __rte_always_inline void
ThreadCtrl_Sleep(ThreadCtrl* ctrl)
{
#ifdef NDNDPDK_THREADSLEEP
  if (unlikely(ctrl->nPolls[0] % ThreadCtrl_SleepAdjustEvery == 0) &&
      unlikely(ctrl->sleepFor < ThreadCtrl_SleepMax)) {
    static_assert(ThreadCtrl_SleepMax * ThreadCtrl_SleepMultiply / ThreadCtrl_SleepDivide +
                      ThreadCtrl_SleepAdd <
                    UINT32_MAX,
                  "");
    static_assert(ThreadCtrl_SleepMultiply >= ThreadCtrl_SleepDivide, "");
    static_assert(ThreadCtrl_SleepDivide > 0, "");
    static_assert(ThreadCtrl_SleepAdd >= 0, "");
    uint64_t sleepFor =
      (uint64_t)ctrl->sleepFor * ThreadCtrl_SleepMultiply / ThreadCtrl_SleepDivide;
    if (ThreadCtrl_SleepAdd == 0) {
      sleepFor = RTE_MAX(sleepFor, (uint64_t)ctrl->sleepFor + 1);
    }
    ctrl->sleepFor = RTE_MIN(ThreadCtrl_SleepMax, (uint32_t)sleepFor + ThreadCtrl_SleepAdd);
  }
  struct timespec req = { .tv_sec = 0, .tv_nsec = ctrl->sleepFor };
  nanosleep(&req, NULL);
#else
  rte_pause();
#endif // NDNDPDK_THREADSLEEP
}

static __rte_always_inline void
ThreadCtrl_SleepReset(ThreadCtrl* ctrl)
{
#ifdef NDNDPDK_THREADSLEEP
  ctrl->sleepFor = ThreadCtrl_SleepMin;
#endif // NDNDPDK_THREADSLEEP
}

/**
 * @brief Determine if a thread loop should continue.
 * @param ctrl ThreadCtrl object reference.
 * @param count integral lvalue, number of processed items in last iteration.
 * @retval true the thread should continue execution.
 * @retval false the thread should stop.
 * @post count==0
 */
#define ThreadCtrl_Continue(ctrl, count)                                                           \
  __extension__({                                                                                  \
    ++(ctrl).nPolls[(int)((count) > 0)];                                                           \
    (ctrl).items += (count);                                                                       \
    if (count == 0) {                                                                              \
      ThreadCtrl_Sleep(&(ctrl));                                                                   \
    } else {                                                                                       \
      ThreadCtrl_SleepReset(&(ctrl));                                                              \
    }                                                                                              \
    (count) = 0;                                                                                   \
    atomic_load_explicit(&(ctrl).stop, memory_order_acquire);                                      \
  })

__attribute__((nonnull)) static inline void
ThreadCtrl_Init(ThreadCtrl* ctrl)
{
  ctrl->sleepFor = ThreadCtrl_SleepMin;
  atomic_init(&ctrl->stop, true);
}

__attribute__((nonnull)) static inline void
ThreadCtrl_RequestStop(ThreadCtrl* ctrl)
{
  atomic_store_explicit(&ctrl->stop, false, memory_order_release);
}

__attribute__((nonnull)) static inline void
ThreadCtrl_FinishStop(ThreadCtrl* ctrl)
{
  atomic_store_explicit(&ctrl->stop, true, memory_order_release);
}

#endif // NDNDPDK_DPDK_THREAD_H
