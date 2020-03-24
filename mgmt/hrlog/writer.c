#include "writer.h"

#include <fcntl.h>
#include <rte_memcpy.h>
#include <sys/mman.h>
#include <sys/types.h>
#include <unistd.h>

struct rte_ring* theHrlogRing = NULL;

int
Hrlog_RunWriter(const char* filename,
                int nSkip,
                int nTotal,
                ThreadStopFlag* stop)
{
  int fd = open(filename, O_RDWR | O_CREAT | O_TRUNC, (mode_t)0600);
  if (fd == -1) {
    return __LINE__;
  }

  HrlogHeader hdr = { .magic = HRLOG_HEADER_MAGIC,
                      .version = HRLOG_HEADER_VERSION,
                      .tschz = rte_get_tsc_hz() };
  if (write(fd, &hdr, sizeof(hdr)) == -1) {
    return __LINE__;
  }

  void* buf[64];
  size_t fileSize = sizeof(hdr) + nTotal * sizeof(buf[0]) + sizeof(buf);
  if (lseek(fd, fileSize - 1, SEEK_SET) == -1) {
    return __LINE__;
  }
  if (write(fd, "", 1) == -1) {
    return __LINE__;
  }

  void** map = mmap(NULL, fileSize, PROT_READ | PROT_WRITE, MAP_SHARED, fd, 0);
  if (map == MAP_FAILED) {
    return __LINE__;
  }
  HrlogEntry* output = RTE_PTR_ADD(map, sizeof(hdr));

  int nCollected = 0;
  while (ThreadStopFlag_ShouldContinue(stop) && nCollected < nTotal) {
    int count = rte_ring_dequeue_burst(theHrlogRing, buf, RTE_DIM(buf), NULL);
    if (unlikely(nSkip > 0)) {
      nSkip -= count;
    } else {
      rte_memcpy(&output[nCollected], buf, count * sizeof(buf[0]));
      nCollected += count;
    }
  }

  if (msync(map, fileSize, MS_SYNC) == -1) {
    return __LINE__;
  }
  if (munmap(map, fileSize) == -1) {
    return __LINE__;
  }
  if (ftruncate(fd, sizeof(hdr) + nCollected * sizeof(buf[0])) == -1) {
    return __LINE__;
  }
  close(fd);
  return 0;
}
