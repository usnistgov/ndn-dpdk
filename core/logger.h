#ifndef NDN_DPDK_CORE_LOGGER_H
#define NDN_DPDK_CORE_LOGGER_H

/// \file

#include "common.h"

#if DOXYGEN
/** \brief Set compile-time maximum log level.
 *
 *  Logging statements below this level incur zero runtime overhead.
 */
#define ZF_LOG_DEF_LEVEL ZF_LOG_VERBOSE
#else
// #define ZF_LOG_DEF_LEVEL ZF_LOG_WARN
#endif

#define ZF_LOG_VERSION_REQUIRED 4
#define ZF_LOG_OUTPUT_LEVEL gZfLogOutputLvl
#define ZF_LOG_SRCLOC ZF_LOG_SRCLOC_SHORT
#include "zf_log.h"

/** \brief Initialize zf_log and set module log level.
 *
 *  This macro must appear in every .c that uses logging.
 *  It is permitted to reuse the same module name in multiple .c files.
 *  \code
 *  INIT_ZF_LOG(MyModule);
 *  \endcode
 *
 *  To specify runtime log level for 'MyModule', pass environment variable:
 *  LOG_MyModule=WARN
 *
 *  To specify runtime log level for all modules, pass environment variable:
 *  LOG=WARN
 *
 *  Acceptable log levels are: VERBOSE, DEBUG, INFO, WARN, ERROR, FATAL, NONE.
 *  These are case sensitive and must be written as upper case.
 *  The default is INFO.
 */
#define INIT_ZF_LOG(module)                                                    \
  static int gZfLogOutputLvl;                                                  \
  RTE_INIT(InitLogOutputLvl) { gZfLogOutputLvl = ParseLogLevel(#module); }     \
  struct __AllowTrailingSemicolon

int ParseLogLevel(const char* module);

#endif // NDN_DPDK_CORE_LOGGER_H
