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
  uint32_t nonce; ///< downstream's nonce
  FaceID face;
  uint8_t congMark;
  bool canBePrefix; ///< Interest has CanBePrefix?
  LpPitToken token; ///< downstream's token
} __rte_aligned(64) PitDn;
static_assert(sizeof(PitDn) <= 64, "");

#endif // NDNDPDK_PCCT_PIT_DN_H
