#ifndef NDNDPDK_TGCONSUMER_COMMON_H
#define NDNDPDK_TGCONSUMER_COMMON_H

/** @file */

#include "../iface/common.h"
#include "enum.h"

enum {
  TgcSeqNumSize = 1 + 1 + sizeof(uint64_t),

  TgcTokenLength = 10,
  TgcTokenOffsetPatternID = 0,
  TgcTokenOffsetRunNum = 1,
  TgcTokenOffsetTimestamp = 2,
};

__attribute__((nonnull)) static __rte_always_inline void
TgcToken_Set(LpPitToken* token, uint8_t patternID, uint8_t runNum, TscTime timestamp) {
  *token = (LpPitToken){
    .length = TgcTokenLength,
  };
  token->value[TgcTokenOffsetPatternID] = patternID;
  token->value[TgcTokenOffsetRunNum] = runNum;
  *(unaligned_uint64_t*)RTE_PTR_ADD(token->value, TgcTokenOffsetTimestamp) = timestamp;
}

__attribute__((nonnull)) static __rte_always_inline uint8_t
TgcToken_GetPatternID(const LpPitToken* token) {
  return token->value[TgcTokenOffsetPatternID];
}

__attribute__((nonnull)) static __rte_always_inline uint8_t
TgcToken_GetRunNum(const LpPitToken* token) {
  return token->value[TgcTokenOffsetRunNum];
}

__attribute__((nonnull)) static __rte_always_inline TscTime
TgcToken_GetTimestamp(const LpPitToken* token) {
  return *(const unaligned_uint64_t*)RTE_PTR_ADD(token->value, TgcTokenOffsetTimestamp);
}

#endif // NDNDPDK_TGCONSUMER_COMMON_H
