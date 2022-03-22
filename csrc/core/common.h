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

#include <rte_bitops.h>
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

#include <spdk/util.h>

#ifdef NDEBUG
#define NDNDPDK_ASSERT(x) RTE_SET_USED(x)
#else
#define NDNDPDK_ASSERT(x) RTE_VERIFY(x)
#endif

#endif // __BPF__

#define STATIC_ASSERT_FUNC_TYPE(typ, func)                                                         \
  static_assert(__builtin_types_compatible_p(typ, typeof(&func)), "")

#ifdef NDEBUG
#define NULLize(x) RTE_SET_USED(x)
#else
/** @brief Set x to NULL to crash on memory access bugs. */
#define NULLize(x)                                                                                 \
  do {                                                                                             \
    (x) = NULL;                                                                                    \
  } while (false)
#endif

#define CLAMP(x, min, max) RTE_MAX((min), RTE_MIN((x), (max)))

#endif // NDNDPDK_CORE_COMMON_H
