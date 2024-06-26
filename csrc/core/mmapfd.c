#include "mmapfd.h"
#include "logger.h"

#include <fcntl.h>
#include <sys/mman.h>
#include <sys/types.h>
#include <unistd.h>

N_LOG_INIT(MmapFd);

#define MmapFd_Error(func) N_LOGE("%s(%s,fd=%d)" N_LOG_ERROR_ERRNO, #func, filename, m->fd, errno)

__attribute__((nonnull)) static inline void
MmapFd_Free(MmapFd* m) {
  close(m->fd);
  *m = (const MmapFd){.fd = -1};
}

bool
MmapFd_Open(MmapFd* m, const char* filename, size_t size) {
  NDNDPDK_ASSERT(size > 0);
  *m = (const MmapFd){
    .size = size,
  };

  m->fd = open(filename, O_RDWR | O_CREAT | O_TRUNC, (mode_t)0644);
  if (m->fd == -1) {
    MmapFd_Error(open);
    return false;
  }
  if (fallocate(m->fd, 0, 0, size) != 0) {
    MmapFd_Error(fallocate);
    if (ftruncate(m->fd, size) == 0) {
      N_LOGW("ftruncate succeeded in place of fallocate, this may affect write performance");
    } else {
      MmapFd_Error(ftruncate);
      goto FAIL;
    }
  }

  m->map = mmap(NULL, size, PROT_READ | PROT_WRITE, MAP_SHARED, m->fd, 0);
  if (m->map == MAP_FAILED) {
    MmapFd_Error(mmap);
    goto FAIL;
  }

  return true;

FAIL:
  MmapFd_Free(m);
  return false;
}

bool
MmapFd_Close(MmapFd* m, const char* filename, size_t size) {
  NDNDPDK_ASSERT(m->size > 0);

  bool ok = false;

  if (msync(m->map, m->size, MS_SYNC) != 0) {
    MmapFd_Error(msync);
    goto FAIL;
  }
  if (munmap(m->map, m->size) != 0) {
    MmapFd_Error(munmap);
    goto FAIL;
  }
  if (ftruncate(m->fd, size) != 0) {
    MmapFd_Error(ftruncate);
    goto FAIL;
  }

  ok = true;
FAIL:
  MmapFd_Free(m);
  return ok;
}
