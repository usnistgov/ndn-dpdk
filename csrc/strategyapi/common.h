#ifndef NDN_DPDK_STRATEGYAPI_COMMON_H
#define NDN_DPDK_STRATEGYAPI_COMMON_H

/// \file

#include "../core/common1.h"

typedef uint64_t TscTime;
typedef int64_t TscDuration;

typedef uint16_t FaceId;
#define FACEID_INVALID 0

#if !__has_attribute(always_inline)
#error always_inline attribute is required
#endif

#define SUBROUTINE __attribute__((always_inline)) static inline

#endif // NDN_DPDK_STRATEGYAPI_COMMON_H
