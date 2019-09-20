#ifndef NDN_DPDK_STRATEGY_API_H
#define NDN_DPDK_STRATEGY_API_H

/// \file

#include "api-fib.h"
#include "api-packet.h"
#include "api-pit.h"

/** \brief Indicate why the strategy program is invoked.
 */
typedef enum SgEvent
{
  SGEVT_NONE,
  SGEVT_TIMER,    ///< timer expires
  SGEVT_INTEREST, ///< Interest arrives
  SGEVT_DATA,     ///< Data arrives
  SGEVT_NACK,     ///< Nack arrives
} SgEvent;

/** \brief Context of strategy invocation.
 */
typedef struct SgCtx
{
  /** \brief Why strategy is triggered.
   */
  SgEvent eventKind;

  /** \brief A bitmask filter on which FIB nexthops should be used.
   *  \warning Not available in \c SGEVT_TIMER and \c SGEVT_DATA and \c SGEVT_NACK.
   */
  SgFibNexthopFilter nhFlt;

  /** \brief Incoming packet.
   *  \warning Not available in \c SGEVT_TIMER.
   */
  const SgPacket* pkt;

  /** \brief FIB entry.
   */
  const SgFibEntry* fibEntry;

  /** \brief PIT entry.
   */
  SgPitEntry* pitEntry;
} SgCtx;

/** \brief Iterate over FIB nexthops passing ctx->nhFlt.
 *  \sa SgFibNexthopIt
 */
inline void
SgFibNexthopIt_Init2(SgFibNexthopIt* it, const SgCtx* ctx)
{
  SgFibNexthopIt_Init(it, ctx->fibEntry, ctx->nhFlt);
}

/** \brief Access FIB entry scratch area as T* type.
 */
#define SgCtx_FibScratchT(ctx, T)                                              \
  __extension__({                                                              \
    static_assert(sizeof(T) <= SG_FIB_DYN_SCRATCH, "");                        \
    (T*)ctx->fibEntry->dyn->scratch;                                           \
  })

/** \brief Access PIT entry scratch area as T* type.
 */
#define SgCtx_PitScratchT(ctx, T)                                              \
  __extension__({                                                              \
    static_assert(sizeof(T) <= SG_PIT_ENTRY_SCRATCH, "");                      \
    (T*)ctx->pitEntry->scratch;                                                \
  })

/** \brief Set a timer to invoke strategy after a duration.
 *  \param afterMillis duration in milliseconds, cannot exceed PIT entry expiration time.
 *  \warning Not available in \c SGEVT_DATA.
 *
 *  Strategy program will be invoked again with \c SGEVT_TIMER after \p after.
 *  However, the timer would be cancelled if strategy program is invoked for any other event,
 *  a different timer is set, or the strategy choice has been changed.
 */
bool
SgSetTimer(SgCtx* ctx, int afterMillis);

typedef enum SgForwardInterestResult
{
  SGFWDI_OK,
  SGFWDI_BADFACE,    ///< face is down or FaceId is invalid
  SGFWDI_ALLOCERR,   ///< allocation error
  SGFWDI_NONONCE,    ///< upstream has rejected all nonces
  SGFWDI_SUPPRESSED, ///< forwarding is suppressed
  SGFWDI_HOPZERO,    ///< HopLimit has become zero
} SgForwardInterestResult;

/** \brief Forward an Interest to a nexthop.
 *  \warning Not available in \c SGEVT_DATA.
 */
SgForwardInterestResult
SgForwardInterest(SgCtx* ctx, FaceId nh);

/** \brief Return Nacks downstream and erase PIT entry.
 *  \warning Only available in \c SGEVT_INTEREST.
 */
void
SgReturnNacks(SgCtx* ctx, SgNackReason reason);

/** \brief The strategy program.
 *  \return status code, ignored by forwarding but appears in logs.
 *
 *  Every strategy must implement this function.
 */
uint64_t
SgMain(SgCtx* ctx);

#endif // NDN_DPDK_STRATEGY_API_H
