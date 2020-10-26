#ifndef NDNDPDK_CORE_LOGGER_H
#define NDNDPDK_CORE_LOGGER_H

/** @file */

#include "common.h"

#ifndef ZF_LOG_DEF_LEVEL
/**
 * @brief Set compile-time maximum log level.
 *
 * Logging statements below this level incur zero runtime overhead.
 */
#define ZF_LOG_DEF_LEVEL ZF_LOG_VERBOSE
#endif

#define ZF_LOG_VERSION_REQUIRED 4
#define ZF_LOG_OUTPUT_LEVEL gZfLogOutputLvl
#define ZF_LOG_SRCLOC ZF_LOG_SRCLOC_SHORT
#include "../vendor/zf_log.h"

/**
 * @brief Initialize zf_log and set module log level.
 * @param module log module name; cannot exceed 16 characters.
 *
 * This macro must appear in every .c that uses logging.
 * It is permitted to reuse the same module name in multiple .c files.
 * @code
 * INIT_ZF_LOG(Foo);
 * @endcode
 */
#define INIT_ZF_LOG(module)                                                                        \
  static int gZfLogOutputLvl;                                                                      \
  RTE_INIT(InitLogOutputLvl)                                                                       \
  {                                                                                                \
    gZfLogOutputLvl = Logger_GetLevel(#module);                                                    \
  }                                                                                                \
  struct AllowTrailingSemicolon_

int
Logger_GetLevel(const char* module);

#endif // NDNDPDK_CORE_LOGGER_H
