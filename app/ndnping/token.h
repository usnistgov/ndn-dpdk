#ifndef NDN_DPDK_APP_NDNPING_TOKEN_H
#define NDN_DPDK_APP_NDNPING_TOKEN_H

/// \file

#include "../../dpdk/tsc.h"

/** \brief Precision of timing measurements.
 *
 *  Duration unit is (TSC >> PING_TIMING_PRECISION).
 */
#define PING_TIMING_PRECISION 16

typedef uint64_t PingTime;

static PingTime
PingTime_FromTsc(TscTime t)
{
  return t >> PING_TIMING_PRECISION;
}

static PingTime
PingTime_Now()
{
  return PingTime_FromTsc(rte_get_tsc_cycles());
}

/** \brief Construct a "PIT token" from ndnping client.
 *
 *  The token has 64 bits:
 *  \li 8 bits of patternId.
 *  \li 8 bits of run number, to distinguish packets from different runs.
 *  \li 48 bits of timestamp (see PING_TIMING_PRECISION).
 */
static uint64_t
PingToken_New(uint8_t patternId, uint8_t runNum, PingTime timestamp)
{
  return ((uint64_t)patternId << 56) | ((uint64_t)runNum << 48) |
         (timestamp & 0xFFFFFFFFFFFF);
}

static uint8_t
PingToken_GetPatternId(uint64_t token)
{
  return token >> 56;
}

static uint8_t
PingToken_GetRunNum(uint64_t token)
{
  return token >> 48;
}

static PingTime
PingToken_GetTimestamp(uint64_t token)
{
  return token & 0xFFFFFFFFFFFF;
}

#endif // NDN_DPDK_APP_NDNPING_TOKEN_H
