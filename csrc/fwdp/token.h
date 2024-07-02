#ifndef NDNDPDK_FWDP_TOKEN_H
#define NDNDPDK_FWDP_TOKEN_H

/** @file */

#include "../pcct/pcc-entry.h"

enum {
  FwTokenLength = 7,
#if RTE_BYTE_ORDER == RTE_LITTLE_ENDIAN
  FwTokenOffsetPccToken = 0,
  FwTokenOffsetFwdID = 6,
#else
  FwTokenOffsetPccToken = -1,
  FwTokenOffsetFwdID = 0,
#endif
};
static_assert(FwTokenLength == PccTokenSize + 1, "");
static_assert(offsetof(LpPitToken, value) + FwTokenOffsetPccToken >= 0, "");
static_assert(RTE_SIZEOF_FIELD(LpPitToken, value) >= sizeof(uint64_t), "");

__attribute__((nonnull)) static __rte_always_inline void
FwToken_Set(LpPitToken* token, uint8_t fwdID, uint64_t pccToken) {
  *token = (LpPitToken){0};
  *(unaligned_uint64_t*)RTE_PTR_ADD(token->value, FwTokenOffsetPccToken) = pccToken;
  token->value[FwTokenOffsetFwdID] = fwdID;
  token->length = FwTokenLength;
}

__attribute__((nonnull)) static __rte_always_inline uint8_t
FwToken_GetFwdID(const LpPitToken* token) {
  return token->value[FwTokenOffsetFwdID];
}

__attribute__((nonnull)) static __rte_always_inline uint64_t
FwToken_GetPccToken(const LpPitToken* token) {
  return *(const unaligned_uint64_t*)RTE_PTR_ADD(token->value, FwTokenOffsetPccToken);
}

#endif // NDNDPDK_FWDP_TOKEN_H
