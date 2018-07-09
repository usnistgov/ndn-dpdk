#include "timing.h"

#include <fcntl.h>
#include <sys/mman.h>
#include <sys/types.h>
#include <unistd.h>

struct rte_ring* gTimingRing = NULL;

int
Timing_RunWriter(const char* filename, int nSkip, int nTotal)
{
  int fd = open(filename, O_RDWR | O_CREAT | O_TRUNC, (mode_t)0600);
  if (fd == -1) {
    return __LINE__;
  }

  TimingHeader hdr = { 0 };
  hdr.magic = TIMING_HEADER_MAGIC;
  hdr.version = TIMING_HEADER_VERSION;
  hdr.tschz = rte_get_tsc_hz();
  if (write(fd, &hdr, sizeof(TimingHeader)) == -1) {
    return __LINE__;
  }

  size_t totalSize = nTotal * sizeof(uint64_t);
  if (lseek(fd, totalSize - 1, SEEK_CUR) == -1) {
    return __LINE__;
  }
  if (write(fd, "", 1) == -1) {
    return __LINE__;
  }

  uint64_t* map = mmap(NULL, sizeof(TimingHeader) + totalSize,
                       PROT_READ | PROT_WRITE, MAP_SHARED, fd, 0);
  if (map == MAP_FAILED) {
    return __LINE__;
  }
  uint64_t* output = RTE_PTR_ADD(map, sizeof(TimingHeader));

  uint64_t buffer[32];
  int nCollected = 0;
  while (nCollected < nTotal) {
    unsigned count =
      rte_ring_sc_dequeue_burst(gTimingRing, (void**)buffer, 32, NULL);
    if (unlikely(nSkip > 0)) {
      nSkip -= count;
    } else if (likely(nCollected + 32 < nTotal)) {
      rte_mov256((uint8_t*)&output[nCollected], (const uint8_t*)buffer);
      nCollected += count;
    } else {
      for (unsigned i = 0; i < count && nCollected < nTotal;
           ++i, ++nCollected) {
        output[nCollected] = buffer[i];
      }
    }
  }

  if (msync(map, totalSize, MS_SYNC) == -1) {
    return __LINE__;
  }
  if (munmap(map, totalSize) == -1) {
    return __LINE__;
  }
  close(fd);
  return 0;
}
