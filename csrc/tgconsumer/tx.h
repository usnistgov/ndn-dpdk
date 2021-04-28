#ifndef NDNDPDK_TGCONSUMER_TX_H
#define NDNDPDK_TGCONSUMER_TX_H

/** @file */

#include "common.h"

#include "../dpdk/thread.h"
#include "../iface/face.h"
#include "../vendor/pcg_basic.h"

/** @brief Per-pattern information in traffic generator consumer. */
typedef struct TgcTxPattern
{
  uint64_t nInterests;
  TgcSeqNum seqNum;
  uint32_t seqNumOffset;

  InterestTemplate tpl;
} TgcTxPattern;

/** @brief Traffic generator consumer TX thread. */
typedef struct TgcTx
{
  FaceID face;
  ThreadStopFlag stop;
  uint8_t runNum;
  uint32_t nWeights;
  struct rte_mempool* interestMp; ///< mempool for Interests
  TscDuration burstInterval;      ///< interval between two bursts

  pcg32_random_t trafficRng;
  NonceGen nonceGen;
  uint64_t nAllocError;

  TgcPatternID weight[TgcMaxSumWeight];
  TgcTxPattern pattern[TgcMaxPatterns];
} TgcTx;

__attribute__((nonnull)) int
TgcTx_Run(TgcTx* ct);

#endif // NDNDPDK_TGCONSUMER_TX_H
