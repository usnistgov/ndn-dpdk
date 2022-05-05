#ifndef NDNDPDK_CORE_RUNNING_STAT_H
#define NDNDPDK_CORE_RUNNING_STAT_H

/** @file */

#include "common.h"

/** @brief Facility to compute mean and variance. */
typedef struct RunningStat
{
  uint64_t i;    ///< count of incoming inputs
  uint64_t mask; ///< take sample only if (i & mask) == 0
  uint64_t n;    ///< count of taken samples
  double m1;
  double m2;
} RunningStat;

__attribute__((nonnull)) static __rte_always_inline bool
RunningStat_Push_(RunningStat* s, double x)
{
  ++s->i;
  if (likely((s->i & s->mask) != 0)) {
    return false;
  }

  uint64_t n1 = s->n++;
  double delta = x - s->m1;
  double deltaN = delta / s->n;
  s->m1 += deltaN;
  s->m2 += delta * deltaN * n1;
  return true;
}

/** @brief Add a sample. */
__attribute__((nonnull)) static inline void
RunningStat_Push(RunningStat* s, double x)
{
  RunningStat_Push_(s, x);
}

/** @brief Facility to compute mean and variance, with integer min and max. */
typedef struct RunningStatI
{
  RunningStat s;
  uint64_t min;
  uint64_t max;
} RunningStatI;

/** @brief Add a sample. */
__attribute__((nonnull)) static inline void
RunningStatI_Push(RunningStatI* s, uint64_t x)
{
  if (RunningStat_Push_(&s->s, x)) {
    s->min = RTE_MIN(s->min, x);
    s->max = RTE_MAX(s->max, x);
  }
}

#endif // NDNDPDK_CORE_RUNNING_STAT_H
