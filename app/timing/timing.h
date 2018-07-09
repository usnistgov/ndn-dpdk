#ifndef NDN_DPDK_APP_TIMING_TIMING_H
#define NDN_DPDK_APP_TIMING_TIMING_H

/// \file

#include "entry.h"

#include <rte_lcore.h>
#include <rte_memcpy.h>
#include <rte_ring.h>

extern struct rte_ring* gTimingRing;

/** \brief Record a non-duration value.
 */
static void
Timing_Record(TimingAction act, uint64_t value)
{
  TimingEntry entry = { 0 };
  entry.act = act;
  entry.lcore = rte_lcore_id();
  entry.value = value;

  rte_ring_mp_enqueue(gTimingRing, (void*)*(const uint64_t*)&entry);
}

/** \brief Record duration of an action.
 */
static void
Timing_Post(TimingAction act, TscTime begin)
{
  TscTime now = rte_get_tsc_cycles();
  uint64_t d = now - begin;
  if (unlikely(d >= (1L << 48))) {
    return;
  }

  Timing_Record(act, d);
}

/** \brief Write timing logs to a file.
 *  \param nSkip how many initial entries to discard.
 *  \param nTotal how many entries to collect.
 */
int Timing_RunWriter(const char* filename, int nSkip, int nTotal);

#endif // NDN_DPDK_APP_TIMING_TIMING_H
