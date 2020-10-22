#ifndef NDNDPDK_TGCONSUMER_RX_H
#define NDNDPDK_TGCONSUMER_RX_H

/** @file */

#include "common.h"

#include "../core/running-stat.h"
#include "../dpdk/thread.h"
#include "../iface/pktqueue.h"

/** @brief Per-pattern information in traffic generator consumer. */
typedef struct TgConsumerRxPattern
{
  uint64_t nData;
  uint64_t nNacks;
  RunningStat rtt;
  uint16_t prefixLen;
} TgConsumerRxPattern;

/** @brief traffic generator consumer RX thread. */
typedef struct TgConsumerRx
{
  PktQueue rxQueue;
  ThreadStopFlag stop;
  uint8_t runNum;
  uint16_t nPatterns;
  TgConsumerRxPattern pattern[TGCONSUMER_MAX_PATTERNS];
} TgConsumerRx;

__attribute__((nonnull)) int
TgConsumerRx_Run(TgConsumerRx* cr);

#endif // NDNDPDK_TGCONSUMER_RX_H
