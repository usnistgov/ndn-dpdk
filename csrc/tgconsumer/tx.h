#ifndef NDNDPDK_TGCONSUMER_TX_H
#define NDNDPDK_TGCONSUMER_TX_H

/** @file */

#include "common.h"

#include "../dpdk/thread.h"
#include "../iface/face.h"
#include "../vendor/pcg_basic.h"

typedef struct TgcTx TgcTx;
typedef struct TgcTxPattern TgcTxPattern;

typedef struct TgcTxDigestPattern
{
  PacketMempools dataMp;
  struct rte_mempool* opPool;
  CryptoQueuePair cqp;
  DataGen dataGen;
  LName prefix;
} TgcTxDigestPattern;

typedef uint16_t (*TgcTxPattern_MakeSuffix)(TgcTx* ct, uint8_t patternID, TgcTxPattern* pattern);

__attribute__((nonnull)) uint16_t
TgcTxPattern_MakeSuffix_Digest(TgcTx* ct, uint8_t patternID, TgcTxPattern* pattern);
__attribute__((nonnull)) uint16_t
TgcTxPattern_MakeSuffix_Offset(TgcTx* ct, uint8_t patternID, TgcTxPattern* pattern);
__attribute__((nonnull)) uint16_t
TgcTxPattern_MakeSuffix_Increment(TgcTx* ct, uint8_t patternID, TgcTxPattern* pattern);

/** @brief Per-pattern information in traffic generator consumer. */
struct TgcTxPattern
{
  uint64_t nInterests;
  TgcTxPattern_MakeSuffix makeSuffix;

  uint8_t a_[6];
  uint8_t seqNumT;
  uint8_t seqNumL;
  uint64_t seqNumV;
  uint8_t digestT;
  uint8_t digestL;
  uint8_t digestV[ImplicitDigestLength];

  union
  {
    uint32_t seqNumOffset;
    TgcTxDigestPattern* digest;
  };

  InterestTemplate tpl;
};
static_assert(offsetof(TgcTxPattern, seqNumL) + 1 == offsetof(TgcTxPattern, seqNumV), "");
static_assert(offsetof(TgcTxPattern, seqNumT) + TgcSeqNumSize + ImplicitDigestSize ==
                offsetof(TgcTxPattern, digestV) + RTE_SIZEOF_FIELD(TgcTxPattern, digestV),
              "");

/** @brief Traffic generator consumer TX thread. */
struct TgcTx
{
  ThreadCtrl ctrl;
  uint32_t nWeights;
  FaceID face;
  uint8_t runNum;
  struct rte_mempool* interestMp;
  TscDuration burstInterval; ///< interval between two bursts

  pcg32_random_t trafficRng;
  NonceGen nonceGen;
  uint64_t nAllocError;

  uint8_t weight[TgcMaxSumWeight];
  TgcTxPattern pattern[TgcMaxPatterns];
};

__attribute__((nonnull)) int
TgcTx_Run(TgcTx* ct);

#endif // NDNDPDK_TGCONSUMER_TX_H
