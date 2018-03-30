#ifndef NDN_DPDK_CONTAINER_PCCT_PIT_UP_H
#define NDN_DPDK_CONTAINER_PCCT_PIT_UP_H

/// \file

#include "common.h"

typedef struct PitEntry PitEntry;

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

/** \brief Record that \p nonce is rejected by upstream.
 */
static void
PitUp_AddRejectedNonce(PitUp* up, uint32_t nonce)
{
  for (int i = PIT_UP_MAX_REJ_NONCES - 1; i > 0; --i) {
    up->rejectedNonces[i] = up->rejectedNonces[i - 1];
  }
  up->rejectedNonces[0] = nonce;
}

/** \brief Choose a nonce for TX Interest.
 *  \param[inout] nonce initial suggested nonce suggestion, usually from RX
 *                      Interest or Nack; final nonce selection.
 *  \retval true a valid nonce is found.
 *  \retval false all DN nonces have been rejected.
 */
bool PitUp_ChooseNonce(PitUp* up, PitEntry* entry, TscTime now,
                       uint32_t* nonce);

/** \brief Record Interest transmission.
 *  \param now time used for calculating InterestLifetime.
 *  \param nonce nonce of TX Interest.
 */
static void PitUp_RecordTx(PitUp* up, PitEntry* entry, TscTime now,
                           uint32_t nonce);
// Definition in pit-entry.h to avoid circular dependency.

#endif // NDN_DPDK_CONTAINER_PCCT_PIT_UP_H
