#include "logger.h"
#include <spdk/log.h>

__attribute__((nonnull)) static ssize_t
Logger_Dpdk(void* ctx, const char* buf, size_t size) {
  FILE* output = ctx;
  fprintf(output, "%d %d %u * ", rte_log_cur_msg_logtype(), rte_log_cur_msg_loglevel(),
          rte_lcore_id());
  size_t res = fwrite(buf, size, 1, output);
  fflush(output);
  return res;
}

int
Logger_Dpdk_Init(FILE* output) {
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
Logger_Spdk(int level, const char* restrict file, const int line, const char* restrict func,
            const char* restrict format, va_list args) {
  uint32_t lvl = spdk2dpdkLogLevels[level];
  if (!rte_log_can_log(RTE_LOGTYPE_SPDK, lvl)) {
    return;
  }

  char buf[4096];
  int len = RTE_MIN((int)sizeof(buf) - 1, vsnprintf(buf, sizeof(buf), format, args));
  if (likely(len > 0 && buf[len - 1] == '\n')) {
    buf[len - 1] = '\0';
  }
  rte_log(lvl, RTE_LOGTYPE_SPDK, "%s @%s:%d\n", buf, file, line);
}

void
Logger_HexDump(const uint8_t* b, size_t count) {
  for (size_t i = 0; i < count; ++i) {
    fprintf(stderr, "%02X", b[i]);
  }
  fprintf(stderr, "\n");
  fflush(stderr);
}
