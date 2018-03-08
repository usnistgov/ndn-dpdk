#ifndef NDN_DPDK_CONTAINER_PCCT_PIT_UP_H
#define NDN_DPDK_CONTAINER_PCCT_PIT_UP_H

/// \file

#include "common.h"

#define PIT_UP_MAX_REJ_NONCES 4

/** \brief A PIT upstream record.
 */
typedef struct PitUp
{
  TscTime lastTx; ///< last TX Interest time
  uint32_t nonce; ///< last TX nonce
  FaceId face;
  bool canBePrefix; ///< Interest has CanBePrefix?
  uint8_t nack;     ///< upstream's nack reason
  uint32_t rejectedNonces[PIT_UP_MAX_REJ_NONCES];
} __rte_aligned(32) PitUp;
static_assert(sizeof(PitUp) <= 32, "");

static void
PitUp_Reset(PitUp* up, FaceId face)
{
  memset(up, 0, sizeof(PitUp));
  up->face = face;
}

static void
PitUp_Copy(PitUp* dst, PitUp* src)
{
  rte_mov32((uint8_t*)dst, (const uint8_t*)src);
  src->face = FACEID_INVALID;
}

static bool
PitUp_HasRejectedNonce(PitUp* up, uint32_t nonce)
{
  for (int i = 0; i < PIT_UP_MAX_REJ_NONCES; ++i) {
    if (up->rejectedNonces[i] == nonce) {
      return true;
    }
  }
  return false;
}

static void
PitUp_AddRejectedNonce(PitUp* up, uint32_t nonce)
{
  for (int i = 1; i < PIT_UP_MAX_REJ_NONCES; ++i) {
    up->rejectedNonces[i] == up->rejectedNonces[i - 1];
  }
  up->rejectedNonces[0] = nonce;
}

#endif // NDN_DPDK_CONTAINER_PCCT_PIT_UP_H
