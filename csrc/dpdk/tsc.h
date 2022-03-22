#ifndef NDNDPDK_DPDK_TSC_H
#define NDNDPDK_DPDK_TSC_H

/** @file */

#include "../core/common.h"

/** @brief TSC clock time point. */
typedef uint64_t TscTime;

/** @brief Duration in TscTime unit. */
typedef int64_t TscDuration;

#ifndef __BPF__

/** @brief TSC time units in one second. */
extern uint64_t TscHz;

/** @brief TSC time units in one nanosecond, @c TscHz/1e9 . */
extern double TscGHz;

/** @brief Seconds in one TSC time unit, @c 1/TscHz . */
extern double TscSeconds;

/** @brief Nanoseconds in one TSC time unit, @c 1/TscGHz . */
extern double TscNanos;

extern double TscTimeRefUnixNano_;
extern double TscTimeRefTsc_;

static __rte_always_inline TscTime
TscTime_FromUnixNano(uint64_t n)
{
  double unixNanoSinceRef = n - TscTimeRefUnixNano_;
  double tscSinceRef = unixNanoSinceRef * TscGHz;
  return TscTimeRefTsc_ + tscSinceRef;
}

static __rte_always_inline uint64_t
TscTime_ToUnixNano(TscTime t)
{
  double tscSinceRef = t - TscTimeRefTsc_;
  double unixNanoSinceRef = tscSinceRef * TscNanos;
  return TscTimeRefUnixNano_ + unixNanoSinceRef;
}

/** @brief Convert milliseconds to @c TscDuration. */
static __rte_always_inline TscDuration
TscDuration_FromMillis(int64_t millis)
{
  return millis * TscHz / 1000;
}

/** @brief Convert @c TscDuration to milliseconds. */
static __rte_always_inline int64_t
TscDuration_ToMillis(TscDuration d)
{
  return d * 1000 / TscHz;
}

#endif // __BPF__

#endif // NDNDPDK_DPDK_TSC_H
