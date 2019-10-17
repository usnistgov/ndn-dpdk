#ifndef NDN_DPDK_CORE_RUNNING_STAT_H
#define NDN_DPDK_CORE_RUNNING_STAT_H

/// \file

#include "../common.h"

#include <float.h>
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
  double m1;
  double m2;
} RunningStat;
static_assert(sizeof(RunningStat) <= RTE_CACHE_LINE_SIZE, "");

/** \brief Set sample rate to once every 2^q inputs.
 *  \param q sample rate, must be between 0 and 30.
 */
static inline void
RunningStat_SetSampleRate(RunningStat* s, int q)
{
  assert(q >= 0 && q <= 30);
  s->mask = (1 << q) - 1;
}

/** \brief Clear statistics portion of \p s.
 */
static inline void
RunningStat_Clear(RunningStat* s, bool enableMinMax)
{
  s->n = 0;
  if (enableMinMax) {
    s->min = DBL_MAX;
    s->max = DBL_MIN;
  } else {
    s->min = NAN;
    s->max = NAN;
  }
  s->m1 = 0.0;
  s->m2 = 0.0;
}

static inline void
RunningStat_UpdateMinMax_(RunningStat* s, double x)
{
  s->min = RTE_MIN(s->min, x);
  s->max = RTE_MAX(s->max, x);
}

static inline void
RunningStat_UpdateM_(RunningStat* s, double x)
{
  uint64_t n1 = s->n;
  ++s->n;
  double delta = x - s->m1;
  double deltaN = delta / s->n;
  s->m1 += deltaN;
  s->m2 += delta * deltaN * n1;
}

/** \brief Add a sample to RunningStat with min-max update.
 */
static __rte_always_inline void
RunningStat_Push(RunningStat* s, double x)
{
  ++s->i;
  if (likely((s->i & s->mask) != 0)) {
    return;
  }
  RunningStat_UpdateMinMax_(s, x);
  RunningStat_UpdateM_(s, x);
}

/** \brief Add a sample to RunningStat without min-max update.
 */
static __rte_always_inline void
RunningStat_Push1(RunningStat* s, double x)
{
  ++s->i;
  if (likely((s->i & s->mask) != 0)) {
    return;
  }
  RunningStat_UpdateM_(s, x);
}

#endif // NDN_DPDK_CORE_RUNNING_STAT_H
