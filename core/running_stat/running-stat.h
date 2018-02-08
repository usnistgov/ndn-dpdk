#ifndef NDN_DPDK_CORE_RUNNING_STAT_H
#define NDN_DPDK_CORE_RUNNING_STAT_H

/// \file

#include "../common.h"

/** \brief Facility to compute min, max, mean, and variance.
 */
typedef struct RunningStat
{
  uint64_t n;
  double min;
  double max;
  double oldM;
  double newM;
  double oldS;
  double newS;
} RunningStat;

/** \brief Add a sample to RunningStat.
 */
static inline void
RunningStat_Push(RunningStat* s, double x)
{
  ++s->n;

  if (unlikely(s->n == 1)) {
    s->min = s->max = s->oldM = s->newM = x;
    s->oldS = 0.0;
  } else {
    s->min = s->min > x ? x : s->min;
    s->max = s->max < x ? x : s->max;
    s->newM = s->oldM + (x - s->oldM) / s->n;
    s->newS = s->oldS + (x - s->oldM) * (x - s->newM);
    s->oldM = s->newM;
    s->oldS = s->newS;
  }
}

#endif // NDN_DPDK_CORE_RUNNING_STAT_H
