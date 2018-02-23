#ifndef NDN_DPDK_DPDK_TSC_H
#define NDN_DPDK_DPDK_TSC_H

/// \file

#include "../core/common.h"
#include <rte_cycles.h>

/** \brief TSC clock time point.
 */
typedef uint64_t TscTime;

/** \brief Duration in TscTime unit.
 */
typedef int64_t TscDuration;

#endif // NDN_DPDK_DPDK_TSC_H
