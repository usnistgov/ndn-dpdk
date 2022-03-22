#ifndef NDNDPDK_PCCT_PIT_UP_H
#define NDNDPDK_PCCT_PIT_UP_H

/** @file */

#include "../iface/faceid.h"
#include "pit-suppress-config.h"

typedef struct PitEntry PitEntry;

#define PIT_UP_MAX_REJ_NONCES 6

/** @brief A PIT upstream record. */
typedef struct PitUp
{
  uint32_t nonce;   ///< nonce on last sent Interest
  FaceID face;      ///< the upstream face
  bool canBePrefix; ///< sent Interest has CanBePrefix?
  uint8_t nack;     ///< Nack reason against last Interest

  TscTime lastTx;       ///< when last Interest was sent
  TscDuration suppress; ///< suppression duration since lastTx
  uint16_t nTx;         ///< how many Interests were sent
  uint8_t nexthopIndex; ///< FIB nexthop index

  /// nonces rejected by Nack~Duplicate from upstream
  uint32_t rejectedNonces[PIT_UP_MAX_REJ_NONCES];
} __rte_cache_aligned PitUp;
static_assert(sizeof(PitUp) <= RTE_CACHE_LINE_SIZE, "");

__attribute__((nonnull)) static inline void
PitUp_Reset(PitUp* up, FaceID face)
{
  *up = (const PitUp){ .face = face };
}

/** @brief Determine if forwarding should be suppressed. */
__attribute__((nonnull)) static inline bool
PitUp_ShouldSuppress(PitUp* up, TscTime now)
{
  return up->lastTx + up->suppress > now;
}

/** @brief Record that @p nonce is rejected by upstream. */
__attribute__((nonnull)) static inline void
PitUp_AddRejectedNonce(PitUp* up, uint32_t nonce)
{
  memmove(&up->rejectedNonces[1], &up->rejectedNonces[0],
          sizeof(up->rejectedNonces) - sizeof(up->rejectedNonces[0]));
  up->rejectedNonces[0] = nonce;
}

/**
 * @brief Choose a nonce for TX Interest.
 * @param[inout] nonce initial suggested nonce suggestion, usually from RX
 *                     Interest or Nack; final nonce selection.
 * @retval true a valid nonce is found.
 * @retval false all DN nonces have been rejected.
 */
__attribute__((nonnull)) bool
PitUp_ChooseNonce(PitUp* up, PitEntry* entry, TscTime now, uint32_t* nonce);

/**
 * @brief Record Interest transmission.
 * @param now time used for calculating InterestLifetime.
 * @param nonce nonce of TX Interest.
 */
__attribute__((nonnull)) void
PitUp_RecordTx(PitUp* up, PitEntry* entry, TscTime now, uint32_t nonce,
               const PitSuppressConfig* suppressCfg);

#endif // NDNDPDK_PCCT_PIT_UP_H
