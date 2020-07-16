/**
 * @mainpage
 *
 * https://github.com/usnistgov/ndn-dpdk
 */

#ifndef NDNDPDK_CORE_COMMON_H
#define NDNDPDK_CORE_COMMON_H

/** @file */

#include <assert.h>
#include <inttypes.h>
#include <limits.h>
#include <memory.h>
#include <stdatomic.h>
#include <stdbool.h>
#include <stddef.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include <rte_config.h>

#include <rte_branch_prediction.h>
#include <rte_common.h>

#ifndef __BPF__

#include <float.h>
#include <math.h>
#include <sys/queue.h>

#include <rte_cycles.h>
#include <rte_debug.h>
#include <rte_errno.h>
#include <rte_malloc.h>

#ifdef NDEBUG
#define NDNDPDK_ASSERT(x) RTE_SET_USED(x)
#else
#define NDNDPDK_ASSERT(x) RTE_VERIFY(x)
#endif

#endif // __BPF__

/** @brief Compute ceil( @p a / @p b ) . */
#define DIV_CEIL(a, b) (((a) + (b)-1) / (b))

#endif // NDNDPDK_CORE_COMMON_H
