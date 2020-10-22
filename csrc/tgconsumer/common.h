#ifndef NDNDPDK_TGCONSUMER_COMMON_H
#define NDNDPDK_TGCONSUMER_COMMON_H

/** @file */

#include "../core/common.h"

#define TGCONSUMER_MAX_PATTERNS 256
#define TGCONSUMER_MAX_SUM_WEIGHT 32768
#define TGCONSUMER_TX_BURST_SIZE 64

#define TGCONSUMER_SUFFIX_LEN 10 // T+L+sizeof(uint64)

typedef uint8_t PingPatternId;
static_assert(UINT8_MAX <= (TGCONSUMER_MAX_PATTERNS - 1), "");

#endif // NDNDPDK_TGCONSUMER_COMMON_H
