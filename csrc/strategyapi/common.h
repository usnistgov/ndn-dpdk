#ifndef NDNDPDK_STRATEGYAPI_COMMON_H
#define NDNDPDK_STRATEGYAPI_COMMON_H

/** @file */

#include "../dpdk/tsc.h"

typedef uint16_t FaceID;

#if !__has_attribute(always_inline)
#error always_inline attribute is required
#endif

/**
 * @brief Indicate that a function is a subroutine.
 *
 * uBPF cannot resolve internal CALL instructions. Thus, every subroutine must be marked inline
 * with this macro to ensure it does not compile into a CALL instruction.
 */
#define SUBROUTINE __rte_always_inline

#endif // NDNDPDK_STRATEGYAPI_COMMON_H
