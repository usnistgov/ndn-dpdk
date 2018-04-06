#ifndef NDN_DPDK_STRATEGY_API_H
#define NDN_DPDK_STRATEGY_API_H

/// \file

#include "api-pit.h"

/** \brief Indicate why the strategy program is invoked.
 */
typedef enum SgEvent {
  SGEVT_NONE,
  SGEVT_TIMER,    ///< timer expires
  SGEVT_INTEREST, ///< Interest arrives
} SgEvent;

/** \brief Context of strategy invocation.
 */
typedef struct SgCtx
{
  SgEvent eventKind;
  SgPitEntry* pitEntry;
  FaceId* nexthops;
  uint8_t nNexthops;
} SgCtx;

/** \brief Set a timer to invoke strategy after a duration.
 *
 *  \c Program will be invoked again with \c SGEVT_TIMER after \p after.
 *  However, the timer would be cancelled if \c Program is invoked for any other event,
 *  a different timer is set, or the strategy choice has been changed.
 */
void SgSetTimer(SgCtx* ctx, TscDuration after);

typedef enum SgForwardInterestResult {
  SGFWDI_OK,
  SGFWDI_BADFACE,    ///< FaceId is invalid
  SGFWDI_ALLOCERR,   ///< allocation error
  SGFWDI_NONONCE,    ///< upstream has rejected all nonces
  SGFWDI_SUPPRESSED, ///< forwarding is suppressed
} SgForwardInterestResult;

/** \brief Forward an Interest to a nexthop.
 */
SgForwardInterestResult SgForwardInterest(SgCtx* ctx, FaceId nh);

/** \brief Return Nacks downstream and erase PIT entry.
 */
void SgReturnNacks(SgCtx* ctx);

/** \brief The strategy program.
 *  \return status code, ignored by forwarding but appears in logs.
 *
 *  Every strategy must implement this function.
 */
uint64_t SgMain(SgCtx* ctx);

#endif // NDN_DPDK_STRATEGY_API_H
