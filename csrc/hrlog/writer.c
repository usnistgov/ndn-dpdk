#include "writer.h"
#include "../core/mmapfd.h"
#include "../dpdk/tsc.h"

struct rte_ring* theHrlogRing = NULL;

bool
Hrlog_RunWriter(const char* filename, int nSkip, int nTotal, ThreadStopFlag* stop)
{
  HrlogHeader hdr = { .magic = HRLOG_HEADER_MAGIC,
                      .version = HRLOG_HEADER_VERSION,
                      .tschz = TscHz };
  void* buf[64];

  MmapFd m;
  if (!MmapFd_Open(&m, filename, sizeof(hdr) + nTotal * sizeof(buf[0]) + sizeof(buf))) {
    return false;
  }

  rte_memcpy(MmapFd_At(&m, 0), &hdr, sizeof(hdr));
  HrlogEntry* output = MmapFd_At(&m, sizeof(hdr));

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

  return MmapFd_Close(&m, sizeof(hdr) + nCollected * sizeof(buf[0]));
}
