#include "mmapfd.h"
#include "logger.h"

#include <fcntl.h>
#include <sys/mman.h>
#include <sys/types.h>
#include <unistd.h>

N_LOG_INIT(MmapFd);

#define MmapFd_Error(func)                                                                         \
  N_LOGE("%s(%s,fd=%d)" N_LOG_ERROR_ERRNO, #func, m->filename, m->fd, errno)

bool
MmapFd_Open(MmapFd* m, const char* filename, size_t size)
{
  NDNDPDK_ASSERT(size > 0);
  *m = (const MmapFd){
    .filename = strdup(filename),
    .size = size,
  };

  m->fd = open(filename, O_RDWR | O_CREAT | O_TRUNC, (mode_t)0644);
  if (m->fd == -1) {
    MmapFd_Error(open);
    return false;
  }

  if (lseek(m->fd, size - 1, SEEK_SET) == -1) {
    MmapFd_Error(lseek);
    goto FAIL;
  }
  if (write(m->fd, "", 1) == -1) {
    MmapFd_Error(write);
    goto FAIL;
  }

  m->map = mmap(NULL, size, PROT_READ | PROT_WRITE, MAP_SHARED, m->fd, 0);
  if (m->map == MAP_FAILED) {
    MmapFd_Error(mmap);
    goto FAIL;
  }

  return true;

FAIL:
  close(m->fd);
  return false;
}

bool
MmapFd_Close(MmapFd* m, size_t size)
{
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
  free((void*)m->filename);
  close(m->fd);
  *m = (const MmapFd){ 0 };
  return ok;
}
