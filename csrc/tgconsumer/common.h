#ifndef NDNDPDK_TGCONSUMER_COMMON_H
#define NDNDPDK_TGCONSUMER_COMMON_H

/** @file */

#include "../iface/common.h"
#include "enum.h"

/** @brief Sequence number component. */
typedef struct TgcSeqNum
{
  char a_[6]; // make compV aligned
  uint8_t compT;
  uint8_t compL;
  uint64_t compV; ///< sequence number in native endianness
} TgcSeqNum;
static_assert(offsetof(TgcSeqNum, compV) % sizeof(uint64_t) == 0, "");

#define TGCONSUMER_SEQNUM_SIZE (sizeof(TgcSeqNum) - offsetof(TgcSeqNum, compT))

enum
{
  TgcTokenLength = 10,
  TgcTokenOffsetPatternID = 0,
  TgcTokenOffsetRunNum = 1,
  TgcTokenOffsetTimestamp = 2,
};

static __rte_always_inline void
TgcToken_Set(LpPitToken* token, uint8_t patternID, uint8_t runNum, TscTime timestamp)
{
  *token = (LpPitToken){
    .length = TgcTokenLength,
  };
  token->value[TgcTokenOffsetPatternID] = patternID;
  token->value[TgcTokenOffsetRunNum] = runNum;
  *(unaligned_uint64_t*)RTE_PTR_ADD(token->value, TgcTokenOffsetTimestamp) = timestamp;
}

static __rte_always_inline uint8_t
TgcToken_GetPatternID(const LpPitToken* token)
{
  return token->value[TgcTokenOffsetPatternID];
}

static __rte_always_inline uint8_t
TgcToken_GetRunNum(const LpPitToken* token)
{
  return token->value[TgcTokenOffsetRunNum];
}

static __rte_always_inline TscTime
TgcToken_GetTimestamp(const LpPitToken* token)
{
  return *(const unaligned_uint64_t*)RTE_PTR_ADD(token->value, TgcTokenOffsetTimestamp);
}

#endif // NDNDPDK_TGCONSUMER_COMMON_H
