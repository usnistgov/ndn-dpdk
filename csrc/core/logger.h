#ifndef NDNDPDK_CORE_LOGGER_H
#define NDNDPDK_CORE_LOGGER_H

/** @file */

#include "common.h"

#ifdef N_LOG_LEVEL
#undef RTE_LOG_DP_LEVEL
#define RTE_LOG_DP_LEVEL N_LOG_LEVEL
#endif

#define N_LOG_INIT(module)                                                                         \
  static int RTE_LOGTYPE_NDN = -1;                                                                 \
  RTE_INIT(Logger_Init_##module)                                                                   \
  {                                                                                                \
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
#define N_LOG_ERROR_STR N_LOG_ERROR("%s")

int
Logger_Dpdk_Init(FILE* output);

void
Logger_Spdk(int level, const char* file, const int line, const char* func, const char* format,
            va_list args);

#endif // NDNDPDK_CORE_LOGGER_H
