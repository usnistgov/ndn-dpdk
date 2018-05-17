#ifndef NDN_DPDK_STRATEGY_API_H
#define NDN_DPDK_STRATEGY_API_H

/// \file

#include "api-fib.h"
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
  SgFibNexthopFilter nhFlt;
  SgFibEntry* fibEntry;
  SgPitEntry* pitEntry;
} SgCtx;

/** \brief Iterator over FIB nexthops that pass ctx->nhFlt.
 *  \param index undeclared variable name for the entry.
 *  \param nh declared FaceId variable for nexthop face.
 *
 *  Example:
 *  \code
 *  FaceId nh;
 *  SgCtx_ForEachNexthop(ctx, i, nh) {
 *    // use i and nh
 *    // 'continue' and 'break' are available
 *  }
 *  \endcode
 */
#define SgCtx_ForEachNexthop(ctx, index, nh)                                   \
  SgFibNexthopFilter_ForEach(ctx->nhFlt, ctx->fibEntry, index, nh)

/** \brief Access FIB entry scratch area as T* type.
 */
#define SgCtx_FibScratchT(ctx, T)                                              \
  __extension__({                                                              \
    static_assert(sizeof(T) <= SG_FIB_DYN_SCRATCH, "");                        \
    (T*)ctx->pitEntry->scratch;                                                \
  })

/** \brief Access PIT entry scratch area as T* type.
 */
#define SgCtx_PitScratchT(ctx, T)                                              \
  __extension__({                                                              \
    static_assert(sizeof(T) <= SG_PIT_ENTRY_SCRATCH, "");                      \
    (T*)ctx->pitEntry->scratch;                                                \
  })

/** \brief Set a timer to invoke strategy after a duration.
 *
 *  \c Program will be invoked again with \c SGEVT_TIMER after \p after.
 *  However, the timer would be cancelled if \c Program is invoked for any other event,
 *  a different timer is set, or the strategy choice has been changed.
 */
void SgSetTimer(SgCtx* ctx, TscDuration after);

typedef enum SgForwardInterestResult {
  SGFWDI_OK,
  SGFWDI_BADFACE,    ///< face is down or FaceId is invalid
  SGFWDI_ALLOCERR,   ///< allocation error
  SGFWDI_NONONCE,    ///< upstream has rejected all nonces
  SGFWDI_SUPPRESSED, ///< forwarding is suppressed
  SGFWDI_HOPZERO,    ///< HopLimit has become zero
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
