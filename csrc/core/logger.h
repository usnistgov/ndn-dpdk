#ifndef NDNDPDK_CORE_LOGGER_H
#define NDNDPDK_CORE_LOGGER_H

/** @file */

#include "common.h"

#undef RTE_LOG_DP_LEVEL
#ifdef N_LOG_LEVEL
#define RTE_LOG_DP_LEVEL N_LOG_LEVEL
#else
#define RTE_LOG_DP_LEVEL RTE_LOG_DEBUG
#endif

#define N_LOG_INIT(module)                                                                         \
  static int RTE_LOGTYPE_NDN = -1;                                                                 \
  RTE_INIT(Logger_Init_##module) {                                                                 \
    RTE_LOGTYPE_NDN = rte_log_register_type_and_pick_level("NDN." #module, RTE_LOG_INFO);          \
  }                                                                                                \
  struct AllowTrailingSemicolon_

#define N_LOG(lvl, fmt, ...) RTE_LOG_DP(lvl, NDN, fmt "\n", ##__VA_ARGS__)

#define N_LOGV(...) N_LOG(DEBUG, __VA_ARGS__)
#define N_LOGD(...) N_LOG(INFO, __VA_ARGS__)
#define N_LOGI(...) N_LOG(NOTICE, __VA_ARGS__)
#define N_LOGW(...) N_LOG(WARNING, __VA_ARGS__)
#define N_LOGE(...) N_LOG(ERR, __VA_ARGS__)

#define N_LOG_ERROR(s) " ERROR={" s "}"
#define N_LOG_ERROR_BLANK N_LOG_ERROR("-")
#define N_LOG_ERROR_ERRNO N_LOG_ERROR("errno<%d>")
#define N_LOG_ERROR_STR N_LOG_ERROR("%s")

__attribute__((nonnull)) int
Logger_Dpdk_Init(FILE* output);

__attribute__((nonnull)) void
Logger_Spdk(int level, const char* restrict file, const int line, const char* restrict func,
            const char* restrict format, va_list args);

/**
 * @brief Print buffer in hexadecimal to stderr.
 *
 * This is only used during debugging, and should not appear in committed code.
 */
__attribute__((nonnull)) void
Logger_HexDump(const uint8_t* b, size_t count);

typedef struct DebugString {
  uint32_t pos;
  uint32_t cap;
  char buffer[8]; // actual capacity is DebugString_MaxCapacity
} DebugString;

/**
 * @brief Obtain a buffer for populating a debug string.
 * @param capacity buffer capacity; panics on oversized request.
 * @returns pointer to a per-lcore static buffer that will be overwritten on subsequent calls
 */
__attribute__((returns_nonnull)) DebugString*
DebugString_Get(size_t capacity);

/** @brief Declare a debug string variable within a function. */
#define DebugString_Use(capacity) DebugString* myDebugString = DebugString_Get((capacity))

/**
 * @brief Append to the local debug string variable.
 * @param fn sprintf-like function.
 */
#define DebugString_Append(fn, ...)                                                                \
  do {                                                                                             \
    myDebugString->pos += fn(RTE_PTR_ADD(myDebugString->buffer, myDebugString->pos),               \
                             myDebugString->cap - myDebugString->pos, __VA_ARGS__);                \
  } while (false)

/** @brief Return the buffer in local debug string variable after bounds checking. */
#define DebugString_Return()                                                                       \
  do {                                                                                             \
    NDNDPDK_ASSERT(myDebugString->pos < myDebugString->cap);                                       \
    return myDebugString->buffer;                                                                  \
  } while (false)

#endif // NDNDPDK_CORE_LOGGER_H
