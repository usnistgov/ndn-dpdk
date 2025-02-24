#ifndef NDNDPDK_STRATEGYAPI_API_H
#define NDNDPDK_STRATEGYAPI_API_H

/** @file */

#include "../strategycode/sec.h"
#include "fib.h"
#include "packet.h"
#include "pit.h"

/** @brief Global static parameters. */
typedef struct SgGlobal {
  uint64_t tscHz;
} SgGlobal;

/** @brief Indicate why the strategy program is invoked. */
typedef enum SgEvent {
  SGEVT_NONE,
  SGEVT_INTEREST, ///< Interest arrives
  SGEVT_DATA,     ///< Data arrives
  SGEVT_NACK,     ///< Nack arrives
  SGEVT_TIMER,    ///< timer expires
} __rte_packed SgEvent;

/** @brief Context of strategy invocation. */
typedef struct SgCtx {
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

  /** @brief FIB entry dynamic area. */
  SgFibEntryDyn* fibEntryDyn;

  /** @brief PIT entry. */
  SgPitEntry* pitEntry;
} SgCtx;

/** @brief Convert milliseconds to TscDuration. */
SUBROUTINE TscDuration
SgTscFromMillis(SgCtx* ctx, uint64_t millis) {
  return millis * ctx->global->tscHz / 1000;
}

/**
 * @brief Iterate over FIB nexthops passing ctx->nhFlt.
 * @sa SgFibNexthopIt
 */
SUBROUTINE void
SgFibNexthopIt_InitCtx(SgFibNexthopIt* it, const SgCtx* ctx) {
  SgFibNexthopIt_Init(it, ctx->fibEntry, ctx->nhFlt);
}

/** @brief Access FIB entry scratch area as T* type. */
#define SgCtx_FibScratchT(ctx, T)                                                                  \
  __extension__({                                                                                  \
    static_assert(sizeof(T) <= FibScratchSize, "");                                                \
    (T*)(ctx)->fibEntryDyn->scratch;                                                               \
  })

/** @brief Access PIT entry scratch area as T* type. */
#define SgCtx_PitScratchT(ctx, T)                                                                  \
  __extension__({                                                                                  \
    static_assert(sizeof(T) <= PitScratchSize, "");                                                \
    (T*)(ctx)->pitEntry->scratch;                                                                  \
  })

/**
 * @brief Generate a random integer.
 * @param max exclusive maximum.
 * @return uniformly distributed random number r, where 0 <= r < max .
 */
__attribute__((nonnull)) uint32_t
SgRandInt(SgCtx* ctx, uint32_t max);

/**
 * @brief Set a timer to invoke strategy after a duration.
 * @param after duration in TSC unit, cannot exceed PIT entry expiration time.
 * @pre Not available in @c SGEVT_DATA .
 *
 * Strategy program will be invoked again with @c SGEVT_TIMER after @p after .
 * However, the timer would be cancelled if strategy program is invoked for any other event,
 * a different timer is set, or the strategy choice has been changed.
 */
__attribute__((nonnull)) bool
SgSetTimer(SgCtx* ctx, TscDuration after);

typedef enum SgForwardInterestResult {
  SGFWDI_OK,         ///< success
  SGFWDI_BADFACE,    ///< face is down or FaceID is invalid
  SGFWDI_ALLOCERR,   ///< allocation error
  SGFWDI_NONONCE,    ///< upstream has rejected all nonces
  SGFWDI_SUPPRESSED, ///< forwarding is suppressed
  SGFWDI_HOPZERO,    ///< HopLimit has become zero
} __rte_packed SgForwardInterestResult;

/**
 * @brief Forward an Interest to a nexthop.
 * @pre Not available in @c SGEVT_DATA .
 */
__attribute__((nonnull)) SgForwardInterestResult
SgForwardInterest(SgCtx* ctx, FaceID nh);

/**
 * @brief Return Nacks downstream and erase PIT entry.
 * @pre Only available in @c SGEVT_INTEREST .
 */
__attribute__((nonnull)) void
SgReturnNacks(SgCtx* ctx, NackReason reason);

/**
 * @brief The strategy dataplane program.
 * @return status code, ignored but may appear in logs.
 *
 * Every strategy must implement this function.
 */
__attribute__((section(SGSEC_MAIN), used, nonnull)) uint64_t
SgMain(SgCtx* ctx);

enum {
  SGJSON_SCALAR = -1,
  SGJSON_LEN = -2,
};

/**
 * @brief Retrieve JSON parameter integer value.
 * @param path JSON property path, using '.' separator for nested path.
 *             Due to eBPF loader limitation, this string should be written as a mutable char[]
 *             allocated on stack. A string literal may cause "resolve_xsym(.L.str) error -2".
 * @param index index into JSON array, or @c SGJSON_SCALAR to retrieve scalar value,
 *              or @c SGJSON_LEN to retrieve array length.
 * @param dst destination pointer.
 * @return whether success.
 * @pre Only available in @c SgInit .
 */
__attribute__((nonnull)) bool
SgGetJSON(SgCtx* ctx, const char* path, int index, int64_t* dst);

/**
 * @brief Retrieve JSON parameter integer scalar value.
 * @param path JSON property path, using '.' separator for nested path.
 * @param dflt default value.
 * @return retrieved or default value.
 */
#define SgGetJSONScalar(ctx, path, dflt)                                                           \
  __extension__({                                                                                  \
    int64_t value = (dflt);                                                                        \
    char pathA[] = (path);                                                                         \
    SgGetJSON((ctx), pathA, SGJSON_SCALAR, &value);                                                \
    value;                                                                                         \
  })

/**
 * @brief Retrieve JSON parameter integer array.
 * @param dst destination array.
 * @param path JSON property path, using '.' separator for nested path.
 * @param dflt default value.
 * @return retrieved length.
 */
#define SgGetJSONSlice(dst, ctx, path, dflt)                                                       \
  __extension__({                                                                                  \
    int64_t length = (0);                                                                          \
    char pathA[] = (path);                                                                         \
    SgGetJSON((ctx), pathA, SGJSON_LEN, &length);                                                  \
    for (int64_t i = 0; i < RTE_DIM((dst)); ++i) {                                                 \
      int64_t value = (dflt);                                                                      \
      if (i < length) {                                                                            \
        SgGetJSON((ctx), pathA, i, &value);                                                        \
      }                                                                                            \
      (dst)[i] = value;                                                                            \
    }                                                                                              \
    length;                                                                                        \
  })

/**
 * @brief The strategy initialization procedure.
 * @return status code, ignored but may appear in logs.
 *
 * A strategy should implement this function if it accepts parameters.
 * This is called when a strategy is activated on a FIB entry.
 * It should populate FIB entry scratch area according to JSON parameters.
 */
__attribute__((section(SGSEC_INIT), used, nonnull)) uint64_t
SgInit(SgCtx* ctx);

/**
 * @brief Declare JSON schema.
 *
 * A strategy should provide a JSON schema if it accepts parameters.
 * Input parameters are validated against this schema prior to calling @c SgInit .
 */
#define SGINIT_SCHEMA(...)                                                                         \
  char SgJSONSchema[] __attribute__((section(SGSEC_SCHEMA), used)) = #__VA_ARGS__;

#endif // NDNDPDK_STRATEGYAPI_API_H
