#define _GNU_SOURCE
#include <stdio.h>

#include "logger.h"
#include <spdk/log.h>

static ssize_t
Logger_Dpdk(void* ctx, const char* buf, size_t size)
{
  FILE* output = ctx;
  fprintf(output, "%d %d %u * ", rte_log_cur_msg_logtype(), rte_log_cur_msg_loglevel(),
          rte_lcore_id());
  size_t res = fwrite(buf, size, 1, output);
  fflush(output);
  return res;
}

int
Logger_Dpdk_Init(FILE* output)
{
  cookie_io_functions_t cookieFunc = {
    .write = Logger_Dpdk,
  };
  FILE* fp = fopencookie(output, "w+", cookieFunc);
  if (fp == NULL) {
    return -EBADF;
  }
  return rte_openlog_stream(fp);
}

RTE_LOG_REGISTER(RTE_LOGTYPE_SPDK, SPDK, DEBUG);

static const uint32_t spdk2dpdkLogLevels[] = {
  [SPDK_LOG_ERROR] = RTE_LOG_ERR,     [SPDK_LOG_WARN] = RTE_LOG_WARNING,
  [SPDK_LOG_NOTICE] = RTE_LOG_NOTICE, [SPDK_LOG_INFO] = RTE_LOG_INFO,
  [SPDK_LOG_DEBUG] = RTE_LOG_DEBUG,
};

void
Logger_Spdk(int level, __rte_unused const char* file, __rte_unused const int line,
            __rte_unused const char* func, const char* format, va_list args)
{
  rte_vlog(spdk2dpdkLogLevels[level], RTE_LOGTYPE_SPDK, format, args);
}
