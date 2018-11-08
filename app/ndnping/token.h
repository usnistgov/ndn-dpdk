#ifndef NDN_DPDK_APP_NDNPING_TOKEN_H
#define NDN_DPDK_APP_NDNPING_TOKEN_H

/// \file

#include "../../core/common.h"

/** \brief Precision of timing measurements.
 *
 *  Duration unit is (TSC >> NDNPING_TIMING_PRECISION).
 */
#define NDNPING_TIMING_PRECISION 16

static uint64_t
Ndnping_Now()
{
  return rte_get_tsc_cycles() >> NDNPING_TIMING_PRECISION;
}

/** \brief Construct a "PIT token" from ndnping client.
 *
 *  The token has 64 bits:
 *  \li 8 bits of patternId.
 *  \li 8 bits of zeros.
 *  \li 48 bits of timestamp (see NDNPING_TIMING_PRECISION).
 */
static uint64_t
NdnpingToken_New(uint8_t patternId, uint64_t timestamp)
{
  return ((uint64_t)patternId << 56) | (timestamp & 0xFFFFFFFFFFFF);
}

static uint8_t
NdnpingToken_GetPatternId(uint64_t token)
{
  return token >> 56;
}

static uint64_t
NdnpingToken_GetTimestamp(uint64_t token)
{
  return token & 0xFFFFFFFFFFFF;
}

#endif // NDN_DPDK_APP_NDNPING_TOKEN_H
