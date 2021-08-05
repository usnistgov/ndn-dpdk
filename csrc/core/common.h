/**
 * @mainpage
 *
 * https://github.com/usnistgov/ndn-dpdk
 */

#ifndef NDNDPDK_CORE_COMMON_H
#define NDNDPDK_CORE_COMMON_H

/** @file */

#if __INTELLISENSE__
// https://github.com/microsoft/vscode-cpptools/issues/4503
#pragma diag_suppress 1094
#endif

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

#define ALLOW_EXPERIMENTAL_API
#include <rte_config.h>

#include <rte_branch_prediction.h>
#include <rte_common.h>

#ifndef __BPF__

#include <float.h>
#include <math.h>
#include <sys/queue.h>

#include <rte_byteorder.h>
#include <rte_cycles.h>
#include <rte_debug.h>
#include <rte_errno.h>
#include <rte_malloc.h>
#include <rte_memcpy.h>

#ifdef NDEBUG
#define NDNDPDK_ASSERT(x) RTE_SET_USED(x)
#else
#define NDNDPDK_ASSERT(x) RTE_VERIFY(x)
#endif

#endif // __BPF__

/** @brief Compute ceil( @p a / @p b ) . */
#define DIV_CEIL(a, b) (((a) + (b)-1) / (b))

#ifdef NDEBUG
#define NULLize(x) (void)(x)
#else
/** @brief Set x to NULL to crash on memory access bugs. */
#define NULLize(x)                                                                                 \
  do {                                                                                             \
    (x) = NULL;                                                                                    \
  } while (false)
#endif

#endif // NDNDPDK_CORE_COMMON_H
