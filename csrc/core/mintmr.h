#ifndef NDNDPDK_CORE_MINTMR_H
#define NDNDPDK_CORE_MINTMR_H

/** @file */

#include "../dpdk/tsc.h"

/** @brief Timer on minute scheduler. */
typedef struct MinTmr
{
  struct cds_list_head h;
} MinTmr;

/** @brief Initialize a timer. */
__attribute__((nonnull)) static __rte_always_inline void
MinTmr_Init(MinTmr* tmr)
{
  CDS_INIT_LIST_HEAD(&tmr->h);
}

typedef void (*MinTmrCb)(MinTmr* tmr, uintptr_t ctx);

/** @brief Minute scheduler. */
typedef struct MinSched
{
  TscDuration interval;
  TscTime nextTime;
  MinTmrCb cb;
  uintptr_t ctx;
  uint32_t lastSlot;
  uint32_t slotMask;
  uint32_t nSlots;
  struct cds_list_head slot[];
} MinSched;

/**
 * @brief Create a minute scheduler.
 * @param nSlotBits set the number of time slots to @c 1<<nSlotBits .
 * @param interval duration between executing slots.
 * @param cb callback function when a timer expires.
 */
__attribute__((returns_nonnull)) MinSched*
MinSched_New(int nSlotBits, TscDuration interval, MinTmrCb cb, uintptr_t ctx);

/** @brief Cancel all events. */
__attribute__((nonnull)) void
MinSched_Clear(MinSched* sched);

/** @brief Destroy a minute scheduler. */
__attribute__((nonnull)) void
MinSched_Close(MinSched* sched);

__attribute__((nonnull)) void
MinSched_Trigger_(MinSched* sched, TscTime now);

/** @brief Trigger callback function on expired timers. */
__attribute__((nonnull)) static __rte_always_inline void
MinSched_Trigger(MinSched* sched)
{
  TscTime now = rte_get_tsc_cycles();
  if (now >= sched->nextTime) {
    MinSched_Trigger_(sched, now);
  }
}

/** @brief Calculate the maximum delay allowed in @c MinTmr_After . */
__attribute__((nonnull)) static inline TscDuration
MinSched_GetMaxDelay(MinSched* sched)
{
  return sched->interval * (sched->nSlots - 2);
}

__attribute__((nonnull)) void
MinTmr_Cancel_(MinTmr* tmr);

/** @brief Cancel a timer. */
__attribute__((nonnull)) static __rte_always_inline void
MinTmr_Cancel(MinTmr* tmr)
{
  if (likely(tmr->h.next != NULL && !cds_list_empty(&tmr->h))) {
    MinTmr_Cancel_(tmr);
  }
}

/**
 * @brief Schedule a timer to expire @p after since current time.
 * @param tmr the timer; any previous setting will be cancelled.
 * @param after expiration delay; negative value is changed to zero.
 * @retval false @p after >= MinSched_GetMaxDelay(sched)
 */
__attribute__((nonnull)) bool
MinTmr_After(MinTmr* tmr, TscDuration after, MinSched* sched);

/** @brief Schedule a timer to expire at @p at . */
__attribute__((nonnull)) static inline bool
MinTmr_At(MinTmr* tmr, TscTime at, MinSched* sched)
{
  TscTime now = rte_get_tsc_cycles();
  return MinTmr_After(tmr, at - now, sched);
}

#endif // NDNDPDK_CORE_MINTMR_H
