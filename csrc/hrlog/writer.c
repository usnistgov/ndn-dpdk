#include "writer.h"
#include "../core/mmapfd.h"
#include "../dpdk/tsc.h"

HrlogRingRef theHrlogRing;

int
HrlogWriter_Run(HrlogWriter* w)
{
  HrlogHeader hdr = { .magic = HRLOG_HEADER_MAGIC,
                      .version = HRLOG_HEADER_VERSION,
                      .tschz = TscHz };
  void* buf[64];

  MmapFd m;
  if (!MmapFd_Open(&m, w->filename, sizeof(hdr) + w->count * sizeof(buf[0]) + sizeof(buf))) {
    return 1;
  }

  rte_memcpy(MmapFd_At(&m, 0), &hdr, sizeof(hdr));
  HrlogEntry* output = MmapFd_At(&m, sizeof(hdr));

  struct rte_ring* oldRing = rcu_xchg_pointer(&theHrlogRing.r, w->queue);
  NDNDPDK_ASSERT(oldRing == NULL);

  int64_t nCollected = 0;
  int64_t count = 0;
  while (ThreadCtrl_Continue(w->ctrl, count) && nCollected < w->count) {
    count = (int64_t)rte_ring_dequeue_burst(w->queue, buf, RTE_DIM(buf), NULL);
    rte_memcpy(&output[nCollected], buf, count * sizeof(buf[0]));
    nCollected += count;
  }

  oldRing = rcu_xchg_pointer(&theHrlogRing.r, NULL);
  NDNDPDK_ASSERT(oldRing == w->queue);

  if (!MmapFd_Close(&m, sizeof(hdr) + RTE_MIN(nCollected, w->count) * sizeof(buf[0]))) {
    return 2;
  }
  return 0;
}
