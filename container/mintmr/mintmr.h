#ifndef NDN_DPDK_CONTAINER_MINTMR_MINTMR_H
#define NDN_DPDK_CONTAINER_MINTMR_MINTMR_H

/// \file

#include "../../dpdk/tsc.h"

typedef struct MinTmr MinTmr;

/** \brief Timer on minute scheduler.
 */
struct MinTmr
{
  MinTmr* prev;
  MinTmr* next;
};

typedef void (*MinTmrCallback)(MinTmr* tmr, void* cbarg);

/** \brief Minute scheduler.
 */
typedef struct MinSched
{
  TscDuration interval;
  TscTime nextTime;
  MinTmrCallback cb;
  void* cbarg;
  uint64_t nTriggered; ///< count of triggered events
  uint16_t lastSlot;
  uint16_t slotMask;
  uint16_t nSlots;
  MinTmr slot[0];
} MinSched;

/** \brief Create a minute scheduler.
 *  \param nSlotBits set the number of time slots to (1 << nSlotBits)
 *  \param interval duration between executing slots
 *  \param cb callback function when a timer expires
 */
MinSched*
MinSched_New(int nSlotBits,
             TscDuration interval,
             MinTmrCallback cb,
             void* cbarg);

/** \brief Destroy a minute scheduler.
 */
void
MinSched_Close(MinSched* sched);

void
MinSched_Trigger_(MinSched* sched, TscTime now);

/** \brief Trigger callback function on expired timers.
 */
static __rte_always_inline void
MinSched_Trigger(MinSched* sched)
{
  TscTime now = rte_get_tsc_cycles();
  if (sched->nextTime > now) {
    return;
  }
  MinSched_Trigger_(sched, now);
}

/** \brief Initialize a timer.
 */
static __rte_always_inline void
MinTmr_Init(MinTmr* tmr)
{
  tmr->next = tmr->prev = NULL;
}

/** \brief Calculate the maximum delay allowed in \c MinTmr_After.
 */
static inline TscDuration
MinSched_GetMaxDelay(MinSched* sched)
{
  return sched->interval * (sched->nSlots - 2);
}

void
MinTmr_Cancel_(MinTmr* tmr);

/** \brief Cancel a timer.
 */
static __rte_always_inline void
MinTmr_Cancel(MinTmr* tmr)
{
  if (tmr->next == NULL) {
    return;
  }
  MinTmr_Cancel_(tmr);
}

/** \brief Schedule a timer to expire \p after since current time.
 *  \param tmr the timer; any previous setting will be cancelled.
 *  \param after expiration delay; negative value is changed to zero
 *  \retval false \p after >= MinSched_GetMaxDelay(sched)
 */
bool
MinTmr_After(MinTmr* tmr, TscDuration after, MinSched* sched);

/** \brief Schedule a timer to expire at \p at.
 */
static inline bool
MinTmr_At(MinTmr* tmr, TscTime at, MinSched* sched)
{
  TscTime now = rte_get_tsc_cycles();
  return MinTmr_After(tmr, at - now, sched);
}

#endif // NDN_DPDK_CONTAINER_MINTMR_MINTMR_H
