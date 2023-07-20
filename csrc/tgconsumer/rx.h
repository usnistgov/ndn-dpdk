#ifndef NDNDPDK_TGCONSUMER_RX_H
#define NDNDPDK_TGCONSUMER_RX_H

/** @file */

#include "common.h"

#include "../core/running-stat.h"
#include "../dpdk/thread.h"
#include "../iface/pktqueue.h"

/** @brief Per-pattern information in traffic generator consumer. */
typedef struct TgcRxPattern {
  uint64_t nNacks;
  RunningStatI rtt;
  uint16_t prefixLen;
} TgcRxPattern;

/** @brief Traffic generator consumer RX thread. */
typedef struct TgcRx {
  ThreadCtrl ctrl;
  PktQueue rxQueue;
  uint8_t runNum;
  uint8_t nPatterns;
  TgcRxPattern pattern[TgcMaxPatterns];
} TgcRx;

__attribute__((nonnull)) int
TgcRx_Run(TgcRx* cr);

#endif // NDNDPDK_TGCONSUMER_RX_H
