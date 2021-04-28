#ifndef NDNDPDK_TGCONSUMER_COMMON_H
#define NDNDPDK_TGCONSUMER_COMMON_H

/** @file */

#include "../iface/common.h"
#include "enum.h"

typedef uint8_t TgcPatternID;
static_assert(UINT8_MAX > TgcMaxPatterns, "");
static_assert(TgcMaxPatterns < (1 << TgcTokenPatternBits), "");

/** @brief Sequence number component. */
typedef struct TgcSeqNum
{
  char a_[6]; // make compV aligned
  uint8_t compT;
  uint8_t compL;
  uint64_t compV; ///< sequence number in native endianness
} __rte_packed TgcSeqNum;
static_assert(offsetof(TgcSeqNum, compV) % sizeof(uint64_t) == 0, "");

#define TGCONSUMER_SEQNUM_SIZE (sizeof(TgcSeqNum) - offsetof(TgcSeqNum, compT))

enum
{
  TgcTimeShift = 64 - TgcTokenTimeBits,
};

/** @brief Construct a "PIT token" for traffic generator client. */
static inline uint64_t
TgcToken_New(uint8_t patternID, uint8_t runNum, TscTime timestamp)
{
  static_assert(TgcTokenPatternBits >= 8, "");
  static_assert(TgcTokenRunBits >= 8, "");
  static_assert(TgcTokenPatternBits + TgcTokenRunBits + TgcTokenTimeBits == 64, "");

  return ((uint64_t)patternID << (TgcTokenRunBits + TgcTokenTimeBits)) |
         ((uint64_t)runNum << TgcTokenTimeBits) | ((uint64_t)timestamp >> TgcTimeShift);
}

static inline TgcPatternID
TgcToken_GetPatternID(uint64_t token)
{
  return token >> (TgcTokenRunBits + TgcTokenTimeBits);
}

static inline uint8_t
TgcToken_GetRunNum(uint64_t token)
{
  return token >> TgcTokenTimeBits;
}

static inline TscTime
TgcToken_GetTimestamp(uint64_t token)
{
  return token << TgcTimeShift;
}

#endif // NDNDPDK_TGCONSUMER_COMMON_H
