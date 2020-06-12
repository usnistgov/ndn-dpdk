#ifndef NDN_DPDK_DPDK_EAL_TSC_H
#define NDN_DPDK_DPDK_EAL_TSC_H

/// \file

#include "../../core/common.h"
#include <rte_cycles.h>

/** \brief TSC clock time point.
 */
typedef uint64_t TscTime;

/** \brief Duration in TscTime unit.
 */
typedef int64_t TscDuration;

/** \brief Convert milliseconds to \c TscDuration.
 */
static inline TscDuration
TscDuration_FromMillis(int64_t millis)
{
  return millis * rte_get_tsc_hz() / 1000;
}

/** \brief Convert \c TscDuration to milliseconds.
 */
static inline int64_t
TscDuration_ToMillis(TscDuration d)
{
  return d * 1000 / rte_get_tsc_hz();
}

#endif // NDN_DPDK_DPDK_EAL_TSC_H
