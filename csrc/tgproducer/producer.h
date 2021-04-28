#ifndef NDNDPDK_TGPRODUCER_PRODUCER_H
#define NDNDPDK_TGPRODUCER_PRODUCER_H

/** @file */

#include "../dpdk/thread.h"
#include "../iface/face.h"
#include "../iface/pktqueue.h"
#include "../vendor/pcg_basic.h"
#include "enum.h"

typedef uint8_t TgpReplyID;

typedef struct TgpReply
{
  uint64_t nInterests;
  DataGen dataGen;
  uint8_t kind;
  uint8_t nackReason;
} TgpReply;

/** @brief Per-prefix information in ndnping server. */
typedef struct TgpPattern
{
  LName prefix;
  uint32_t nWeights;
  uint8_t nReplies;
  TgpReplyID weight[TgpMaxSumWeight];
  TgpReply reply[TgpMaxReplies];
  uint8_t prefixBuffer[NameMaxLength];
} TgpPattern;

/** @brief ndnping server. */
typedef struct Tgp
{
  PktQueue rxQueue;
  PacketMempools mp; ///< mempools for Data encoding
  FaceID face;
  uint8_t nPatterns;

  ThreadStopFlag stop;
  uint64_t nNoMatch;
  uint64_t nAllocError;
  pcg32_random_t replyRng;

  TgpPattern pattern[TgpMaxPatterns];
} Tgp;

__attribute__((nonnull)) int
Tgp_Run(Tgp* p);

#endif // NDNDPDK_TGPRODUCER_PRODUCER_H
