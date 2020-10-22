#ifndef NDNDPDK_TGCONSUMER_TOKEN_H
#define NDNDPDK_TGCONSUMER_TOKEN_H

/** @file */

#include "../dpdk/tsc.h"

/**
 * @brief Precision of timing measurements.
 *
 * Duration unit is (TSC >> TGCONSUMER_TIMING_PRECISION).
 */
#define TGCONSUMER_TIMING_PRECISION 16

typedef uint64_t TgTime;

static inline TgTime
TgTime_FromTsc(TscTime t)
{
  return t >> TGCONSUMER_TIMING_PRECISION;
}

static inline TgTime
TgTime_Now()
{
  return TgTime_FromTsc(rte_get_tsc_cycles());
}

/**
 * @brief Construct a "PIT token" from ndnping client.
 *
 * The token has 64 bits:
 * @li 8 bits of patternId.
 * @li 8 bits of run number, to distinguish packets from different runs.
 * @li 48 bits of timestamp (see TGCONSUMER_TIMING_PRECISION).
 */
static inline uint64_t
TgToken_New(uint8_t patternId, uint8_t runNum, TgTime timestamp)
{
  return ((uint64_t)patternId << 56) | ((uint64_t)runNum << 48) | (timestamp & 0xFFFFFFFFFFFF);
}

static inline uint8_t
TgToken_GetPatternId(uint64_t token)
{
  return token >> 56;
}

static inline uint8_t
TgToken_GetRunNum(uint64_t token)
{
  return token >> 48;
}

static inline TgTime
TgToken_GetTimestamp(uint64_t token)
{
  return token & 0xFFFFFFFFFFFF;
}

#endif // NDNDPDK_TGCONSUMER_TOKEN_H
