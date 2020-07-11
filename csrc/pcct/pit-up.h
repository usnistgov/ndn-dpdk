#ifndef NDN_DPDK_PCCT_PIT_UP_H
#define NDN_DPDK_PCCT_PIT_UP_H

/** @file */

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

  /// nonces rejected by Nack~Duplicate from upstream
  uint32_t rejectedNonces[PIT_UP_MAX_REJ_NONCES];
} __rte_aligned(64) PitUp;
static_assert(sizeof(PitUp) == 64, "");

static inline void
PitUp_Reset(PitUp* up, FaceID face)
{
  memset(up, 0, sizeof(PitUp));
  up->face = face;
}

static inline void
PitUp_Copy(PitUp* dst, PitUp* src)
{
  rte_mov64((uint8_t*)dst, (const uint8_t*)src);
  src->face = 0;
}

/**
 * @brief Determine if forwarding should be suppressed.
 */
static inline bool
PitUp_ShouldSuppress(PitUp* up, TscTime now)
{
  return up->lastTx + up->suppress > now;
}

/**
 * @brief Record that @p nonce is rejected by upstream.
 */
static inline void
PitUp_AddRejectedNonce(PitUp* up, uint32_t nonce)
{
  for (int i = PIT_UP_MAX_REJ_NONCES - 1; i > 0; --i) {
    up->rejectedNonces[i] = up->rejectedNonces[i - 1];
  }
  up->rejectedNonces[0] = nonce;
}

/**
 * @brief Choose a nonce for TX Interest.
 * @param[inout] nonce initial suggested nonce suggestion, usually from RX
 *                     Interest or Nack; final nonce selection.
 * @retval true a valid nonce is found.
 * @retval false all DN nonces have been rejected.
 */
bool
PitUp_ChooseNonce(PitUp* up, PitEntry* entry, TscTime now, uint32_t* nonce);

/**
 * @brief Record Interest transmission.
 * @param now time used for calculating InterestLifetime.
 * @param nonce nonce of TX Interest.
 */
void
PitUp_RecordTx(PitUp* up, PitEntry* entry, TscTime now, uint32_t nonce,
               PitSuppressConfig* suppressCfg);

#endif // NDN_DPDK_PCCT_PIT_UP_H
