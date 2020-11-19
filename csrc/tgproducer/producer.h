#ifndef NDNDPDK_TGPRODUCER_PRODUCER_H
#define NDNDPDK_TGPRODUCER_PRODUCER_H

/** @file */

#include "../dpdk/thread.h"
#include "../iface/face.h"
#include "../iface/pktqueue.h"
#include "../vendor/pcg_basic.h"

#define TGPRODUCER_MAX_PATTERNS 256
#define TGPRODUCER_MAX_REPLIES 8
#define TGPRODUCER_MAX_SUM_WEIGHT 256

typedef uint8_t PingReplyId;

typedef enum TgProducerReplyKind
{
  TGPRODUCER_REPLY_DATA,
  TGPRODUCER_REPLY_NACK,
  TGPRODUCER_REPLY_TIMEOUT,
} TgProducerReplyKind;

typedef struct TgProducerReply
{
  uint64_t nInterests;
  DataGen dataGen;
  uint8_t kind;
  uint8_t nackReason;
} TgProducerReply;

/** @brief Per-prefix information in ndnping server. */
typedef struct TgProducerPattern
{
  LName prefix;
  uint16_t nReplies;
  uint16_t nWeights;
  PingReplyId weight[TGPRODUCER_MAX_SUM_WEIGHT];
  TgProducerReply reply[TGPRODUCER_MAX_REPLIES];
  uint8_t prefixBuffer[NameMaxLength];
} TgProducerPattern;

/** @brief ndnping server. */
typedef struct TgProducer
{
  PktQueue rxQueue;
  PacketMempools mp; ///< mempools for Data encoding
  FaceID face;
  uint16_t nPatterns;
  bool wantNackNoRoute; ///< whether to Nack Interests not matching any pattern

  ThreadStopFlag stop;
  uint64_t nNoMatch;
  uint64_t nAllocError;
  pcg32_random_t replyRng;

  TgProducerPattern pattern[TGPRODUCER_MAX_PATTERNS];
} TgProducer;

__attribute__((nonnull)) int
TgProducer_Run(TgProducer* server);

#endif // NDNDPDK_TGPRODUCER_PRODUCER_H
