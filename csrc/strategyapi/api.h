#ifndef NDNDPDK_STRATEGYAPI_API_H
#define NDNDPDK_STRATEGYAPI_API_H

/** @file */

#include "fib.h"
#include "packet.h"
#include "pit.h"

typedef struct SgGlobal
{
  uint64_t tscHz;
} SgGlobal;

/** @brief Indicate why the strategy program is invoked. */
typedef enum SgEvent
{
  SGEVT_NONE,
  SGEVT_INTEREST, ///< Interest arrives
  SGEVT_DATA,     ///< Data arrives
  SGEVT_NACK,     ///< Nack arrives
  SGEVT_TIMER,    ///< timer expires
} SgEvent;

/** @brief Context of strategy invocation. */
typedef struct SgCtx
{
  /** @brief Global static parameters. */
  const SgGlobal* global;

  /** @brief Packet arrival time or current time. */
  TscTime now;

  /** @brief Why strategy is triggered. */
  SgEvent eventKind;

  /** @brief A bitmask filter on which FIB nexthops should be used. */
  SgFibNexthopFilter nhFlt;

  /**
   * @brief Incoming packet.
   * @pre eventKind is SGEVT_DATA or SGEVT_NACK.
   */
  const SgPacket* pkt;

  /** @brief FIB entry. */
  const SgFibEntry* fibEntry;

  /** @brief PIT entry. */
  SgPitEntry* pitEntry;
} SgCtx;

/** @brief Convert milliseconds to TscDuration. */
#define SgTscFromMillis(ctx, millis) ((millis) * (ctx)->global->tscHz / 1000)

/**
 * @brief Iterate over FIB nexthops passing ctx->nhFlt.
 * @sa SgFibNexthopIt
 */
inline void
SgFibNexthopIt_Init2(SgFibNexthopIt* it, const SgCtx* ctx)
{
  SgFibNexthopIt_Init(it, ctx->fibEntry, ctx->nhFlt);
}

/** @brief Access FIB entry scratch area as T* type. */
#define SgCtx_FibScratchT(ctx, T)                                                                  \
  __extension__({                                                                                  \
    static_assert(sizeof(T) <= SG_FIB_ENTRY_SCRATCH, "");                                          \
    (T*)(ctx)->fibEntry->scratch;                                                                  \
  })

/** @brief Access PIT entry scratch area as T* type. */
#define SgCtx_PitScratchT(ctx, T)                                                                  \
  __extension__({                                                                                  \
    static_assert(sizeof(T) <= SG_PIT_ENTRY_SCRATCH, "");                                          \
    (T*)(ctx)->pitEntry->scratch;                                                                  \
  })

/**
 * @brief Set a timer to invoke strategy after a duration.
 * @param after duration in TSC unit, cannot exceed PIT entry expiration time.
 * @warning Not available in @c SGEVT_DATA.
 *
 * Strategy program will be invoked again with @c SGEVT_TIMER after @p after.
 * However, the timer would be cancelled if strategy program is invoked for any other event,
 * a different timer is set, or the strategy choice has been changed.
 */
__attribute__((nonnull)) bool
SgSetTimer(SgCtx* ctx, TscDuration after);

typedef enum SgForwardInterestResult
{
  SGFWDI_OK,
  SGFWDI_BADFACE,    ///< face is down or FaceID is invalid
  SGFWDI_ALLOCERR,   ///< allocation error
  SGFWDI_NONONCE,    ///< upstream has rejected all nonces
  SGFWDI_SUPPRESSED, ///< forwarding is suppressed
  SGFWDI_HOPZERO,    ///< HopLimit has become zero
} SgForwardInterestResult;

/**
 * @brief Forward an Interest to a nexthop.
 * @pre Not available in @c SGEVT_DATA.
 */
__attribute__((nonnull)) SgForwardInterestResult
SgForwardInterest(SgCtx* ctx, FaceID nh);

/**
 * @brief Return Nacks downstream and erase PIT entry.
 * @pre Only available in @c SGEVT_INTEREST.
 */
__attribute__((nonnull)) void
SgReturnNacks(SgCtx* ctx, SgNackReason reason);

/**
 * @brief The strategy program.
 * @return status code, ignored by forwarding but appears in logs.
 *
 * Every strategy must implement this function.
 */
__attribute__((nonnull)) uint64_t
SgMain(SgCtx* ctx);

#endif // NDNDPDK_STRATEGYAPI_API_H
