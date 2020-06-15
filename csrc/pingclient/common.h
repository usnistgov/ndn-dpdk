#ifndef NDN_DPDK_PINGCLIENT_COMMON_H
#define NDN_DPDK_PINGCLIENT_COMMON_H

/// \file

#include "../core/common.h"

#define PINGCLIENT_MAX_PATTERNS 256
#define PINGCLIENT_MAX_SUM_WEIGHT 32768
#define PINGCLIENT_TX_BURST_SIZE 64

#define PINGCLIENT_SUFFIX_LEN 10 // T+L+sizeof(uint64)

typedef uint8_t PingPatternId;
static_assert(UINT8_MAX <= (PINGCLIENT_MAX_PATTERNS - 1), "");

#endif // NDN_DPDK_PINGCLIENT_COMMON_H
