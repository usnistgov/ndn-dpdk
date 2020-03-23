#include "writer.h"

#include <fcntl.h>
#include <rte_memcpy.h>
#include <sys/mman.h>
#include <sys/types.h>
#include <unistd.h>

struct rte_ring* theHrlogRing = NULL;

int
Hrlog_RunWriter(const char* filename, int nSkip, int nTotal)
{
  int fd = open(filename, O_RDWR | O_CREAT | O_TRUNC, (mode_t)0600);
  if (fd == -1) {
    return __LINE__;
  }

  HrlogHeader hdr = { 0 };
  hdr.magic = HRLOG_HEADER_MAGIC;
  hdr.version = HRLOG_HEADER_VERSION;
  hdr.tschz = rte_get_tsc_hz();
  if (write(fd, &hdr, sizeof(HrlogHeader)) == -1) {
    return __LINE__;
  }

  void* buffer[64];
  size_t fileSize =
    sizeof(HrlogHeader) + nTotal * sizeof(uint64_t) + sizeof(buffer);
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
  HrlogEntry* output = RTE_PTR_ADD(map, sizeof(HrlogHeader));

  int nCollected = 0;
  while (nCollected < nTotal) {
    int count =
      rte_ring_dequeue_burst(theHrlogRing, buffer, RTE_DIM(buffer), NULL);
    if (unlikely(nSkip > 0)) {
      nSkip -= count;
    } else {
      rte_memcpy(&output[nCollected], buffer, count * sizeof(buffer[0]));
      nCollected += count;
    }
  }

  if (msync(map, fileSize, MS_SYNC) == -1) {
    return __LINE__;
  }
  if (munmap(map, fileSize) == -1) {
    return __LINE__;
  }
  if (ftruncate(fd, fileSize - sizeof(buffer)) == -1) {
    return __LINE__;
  }
  close(fd);
  return 0;
}
