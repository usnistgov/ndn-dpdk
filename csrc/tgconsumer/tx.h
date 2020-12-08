#ifndef NDNDPDK_TGCONSUMER_TX_H
#define NDNDPDK_TGCONSUMER_TX_H

/** @file */

#include "common.h"

#include "../dpdk/thread.h"
#include "../iface/face.h"
#include "../vendor/pcg_basic.h"

/** @brief Per-pattern information in traffic generator consumer. */
typedef struct TgConsumerTxPattern
{
  uint64_t nInterests;

  struct
  {
    char _padding[6]; // make compV aligned
    uint8_t compT;
    uint8_t compL;
    uint64_t compV;      ///< sequence number in native endianness
  } __rte_packed seqNum; ///< sequence number component

  uint32_t seqNumOffset;

  InterestTemplate tpl;
} TgConsumerTxPattern;

static_assert(offsetof(TgConsumerTxPattern, seqNum.compV) % sizeof(uint64_t) == 0, "");

/** @brief traffic generator consumer TX thread. */
typedef struct TgConsumerTx
{
  FaceID face;
  ThreadStopFlag stop;
  uint8_t runNum;
  uint16_t nWeights;
  struct rte_mempool* interestMp; ///< mempool for Interests
  TscDuration burstInterval;      ///< interval between two bursts

  pcg32_random_t trafficRng;
  NonceGen nonceGen;
  uint64_t nAllocError;

  PingPatternId weight[TGCONSUMER_MAX_SUM_WEIGHT];
  TgConsumerTxPattern pattern[TGCONSUMER_MAX_PATTERNS];
} TgConsumerTx;

__attribute__((nonnull)) int
TgConsumerTx_Run(TgConsumerTx* ct);

#endif // NDNDPDK_TGCONSUMER_TX_H
