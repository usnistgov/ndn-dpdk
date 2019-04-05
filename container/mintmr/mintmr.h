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
__MinSched_Trigger(MinSched* sched, TscTime now);

/** \brief Trigger callback function on expired timers.
 */
static void
MinSched_Trigger(MinSched* sched)
{
  TscTime now = rte_get_tsc_cycles();
  if (sched->nextTime > now) {
    return;
  }
  __MinSched_Trigger(sched, now);
}

/** \brief Initialize a timer.
 */
static void
MinTmr_Init(MinTmr* tmr)
{
  tmr->next = tmr->prev = NULL;
}

/** \brief Calculate the maximum delay allowed in \c MinTmr_After.
 */
static TscDuration
MinSched_GetMaxDelay(MinSched* sched)
{
  return sched->interval * (sched->nSlots - 2);
}

/** \brief Schedule a timer to expire \p after since current time.
 *  \param tmr the timer, must not be running.
 */
bool
MinTmr_After(MinTmr* tmr, TscDuration after, MinSched* sched);

/** \brief Schedule a timer to expire at \p at.
 *  \param tmr the timer, must not be running.
 */
static bool
MinTmr_At(MinTmr* tmr, TscTime at, MinSched* sched)
{
  TscTime now = rte_get_tsc_cycles();
  TscDuration after = at > now ? at - now : 0;
  return MinTmr_After(tmr, after, sched);
}

void
__MinTmr_Cancel(MinTmr* tmr);

/** \brief Cancel a timer.
 */
static void
MinTmr_Cancel(MinTmr* tmr)
{
  if (tmr->next == NULL) {
    return;
  }
  __MinTmr_Cancel(tmr);
}

#endif // NDN_DPDK_CONTAINER_MINTMR_MINTMR_H
