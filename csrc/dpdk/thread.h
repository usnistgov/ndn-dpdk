#ifndef NDNDPDK_DPDK_THREAD_H
#define NDNDPDK_DPDK_THREAD_H

/** @file */

#include "../core/common.h"

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
  struct timespec req = { .tv_sec = 0, .tv_nsec = ctrl->sleepFor };
  nanosleep(&req, NULL);
#else
  rte_pause();
#endif // NDNDPDK_THREADSLEEP
}

/**
 * @brief Determine if a stop has been requested.
 * @param ctrl ThreadCtrl object reference.
 * @param count integral type, number of processed items.
 * @retval false the thread should stop.
 * @retval true the thread should continue execution.
 * @post count==0
 */
#define ThreadCtrl_Continue(ctrl, count)                                                           \
  __extension__({                                                                                  \
    ++(ctrl).nPolls[(int)((count) > 0)];                                                           \
    (ctrl).items += (count);                                                                       \
    if (count == 0) {                                                                              \
      ThreadCtrl_Sleep(&(ctrl));                                                                   \
    }                                                                                              \
    (count) = 0;                                                                                   \
    atomic_load_explicit(&(ctrl).stop, memory_order_acquire);                                      \
  })

static inline void
ThreadCtrl_Init(ThreadCtrl* ctrl)
{
  ctrl->sleepFor = 1;
  atomic_init(&ctrl->stop, true);
}

static inline void
ThreadCtrl_RequestStop(ThreadCtrl* ctrl)
{
  atomic_store_explicit(&ctrl->stop, false, memory_order_release);
}

static inline void
ThreadCtrl_FinishStop(ThreadCtrl* ctrl)
{
  atomic_store_explicit(&ctrl->stop, true, memory_order_release);
}

#endif // NDNDPDK_DPDK_THREAD_H
