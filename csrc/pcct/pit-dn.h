#ifndef NDNDPDK_PCCT_PIT_DN_H
#define NDNDPDK_PCCT_PIT_DN_H

/** @file */

#include "common.h"

/**
 * @brief A PIT downstream record.
 */
typedef struct PitDn
{
  TscTime expiry; ///< expiration time
  uint64_t token; ///< downstream's token
  uint32_t nonce; ///< downstream's nonce
  FaceID face;
  bool congMark;
  bool canBePrefix; ///< Interest has CanBePrefix?
} __rte_aligned(32) PitDn;
static_assert(sizeof(PitDn) == 32, "");

static inline void
PitDn_Copy(PitDn* dst, PitDn* src)
{
  rte_mov32((uint8_t*)dst, (const uint8_t*)src);
  src->face = 0;
}

#endif // NDNDPDK_PCCT_PIT_DN_H
