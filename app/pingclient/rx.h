#ifndef NDN_DPDK_APP_PINGCLIENT_RX_H
#define NDN_DPDK_APP_PINGCLIENT_RX_H

/// \file

#include "common.h"

#include "../../core/running_stat/running-stat.h"
#include "../../dpdk/thread.h"

/** \brief Per-pattern information in ndnping client.
 */
typedef struct PingClientRxPattern
{
  uint64_t nData;
  uint64_t nNacks;
  RunningStat rtt;
  uint16_t prefixLen;
} PingClientRxPattern;

/** \brief ndnping client.
 */
typedef struct PingClientRx
{
  struct rte_ring* rxQueue;
  ThreadStopFlag stop;
  uint8_t runNum;
  uint16_t nPatterns;
  PingClientRxPattern pattern[PINGCLIENT_MAX_PATTERNS];
} PingClientRx;

void
PingClientRx_Run(PingClientRx* cr);

#endif // NDN_DPDK_APP_PINGCLIENT_RX_H
