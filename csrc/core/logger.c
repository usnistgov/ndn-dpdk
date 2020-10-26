#include "logger.h"

#define LOGGER_ENV "NDNDPDK_LOG"

int
Logger_GetLevel(const char* module)
{
  NDNDPDK_ASSERT(strlen(module) <= 16);
  char envKey[32];
  int envKeyLen = snprintf(envKey, sizeof(envKey), "%s_%s", LOGGER_ENV, module);
  NDNDPDK_ASSERT(envKeyLen > 0 && envKeyLen < (int)sizeof(envKey));

  const char* lvl = getenv(envKey);
  if (lvl == NULL) {
    lvl = getenv(LOGGER_ENV);
  }
  if (lvl == NULL) {
    lvl = "";
  }

  switch (lvl[0]) {
    case 'V':
      return ZF_LOG_VERBOSE;
    case 'D':
      return ZF_LOG_DEBUG;
    case 'I':
      return ZF_LOG_INFO;
    case 'W':
      return ZF_LOG_WARN;
    case 'E':
      return ZF_LOG_ERROR;
    case 'F':
      return ZF_LOG_FATAL;
    case 'N':
      return ZF_LOG_NONE;
  }

  return ZF_LOG_INFO;
}
