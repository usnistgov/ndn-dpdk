#ifndef NDN_DPDK_CORE_RUNNING_STAT_H
#define NDN_DPDK_CORE_RUNNING_STAT_H

/// \file

#include "../common.h"
#include <math.h>

/** \brief Facility to compute min, max, mean, and variance.
 */
typedef struct RunningStat
{
  uint32_t i;    ///< count of incoming inputs
  uint32_t mask; ///< take sample only if (i & mask) == 0
  uint64_t n;    ///< count of taken samples
  double min;
  double max;
  double oldM;
  double newM;
  double oldS;
  double newS;
} RunningStat;
static_assert(sizeof(RunningStat) <= RTE_CACHE_LINE_SIZE, "");

/** \brief Set sample rate to once every 2^q inputs.
 *  \param q sample rate, must be between 0 and 30.
 */
static void
RunningStat_SetSampleRate(RunningStat* s, int q)
{
  assert(q >= 0 && q <= 30);
  s->mask = (1 << q) - 1;
}

static __rte_always_inline void
__RunningStat_UpdateMS(RunningStat* s, double x)
{
  s->newM = s->oldM + (x - s->oldM) / s->n;
  s->newS = s->oldS + (x - s->oldM) * (x - s->newM);
  s->oldM = s->newM;
  s->oldS = s->newS;
}

static void
__RunningStat_Update(RunningStat* s, double x)
{
  ++s->n;
  if (unlikely(s->n == 1)) {
    s->min = s->max = s->oldM = s->newM = x;
    s->oldS = 0.0;
  } else {
    s->min = RTE_MIN(s->min, x);
    s->max = RTE_MAX(s->max, x);
    __RunningStat_UpdateMS(s, x);
  }
}

static void
__RunningStat_Update1(RunningStat* s, double x)
{
  ++s->n;
  if (unlikely(s->n == 1)) {
    s->min = s->max = NAN;
    s->oldM = s->newM = x;
    s->oldS = 0.0;
  } else {
    __RunningStat_UpdateMS(s, x);
  }
}

/** \brief Add a sample to RunningStat.
 */
static __rte_always_inline void
RunningStat_Push(RunningStat* s, double x)
{
  ++s->i;
  if (likely((s->i & s->mask) != 0)) {
    return;
  }
  __RunningStat_Update(s, x);
}

/** \brief Add a sample to RunningStat, and disable min-max.
 */
static __rte_always_inline void
RunningStat_Push1(RunningStat* s, double x)
{
  ++s->i;
  if (likely((s->i & s->mask) != 0)) {
    return;
  }
  __RunningStat_Update1(s, x);
}

#endif // NDN_DPDK_CORE_RUNNING_STAT_H
