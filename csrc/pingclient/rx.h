#ifndef NDNDPDK_PINGCLIENT_RX_H
#define NDNDPDK_PINGCLIENT_RX_H

/** @file */

#include "common.h"

#include "../core/running-stat.h"
#include "../dpdk/thread.h"
#include "../iface/pktqueue.h"

/** @brief Per-pattern information in ndnping client. */
typedef struct PingClientRxPattern
{
  uint64_t nData;
  uint64_t nNacks;
  RunningStat rtt;
  uint16_t prefixLen;
} PingClientRxPattern;

/** @brief ndnping client RX thread. */
typedef struct PingClientRx
{
  PktQueue rxQueue;
  ThreadStopFlag stop;
  uint8_t runNum;
  uint16_t nPatterns;
  PingClientRxPattern pattern[PINGCLIENT_MAX_PATTERNS];
} PingClientRx;

__attribute__((nonnull)) int
PingClientRx_Run(PingClientRx* cr);

#endif // NDNDPDK_PINGCLIENT_RX_H
