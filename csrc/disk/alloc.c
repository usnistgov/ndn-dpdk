#include "alloc.h"

DiskAlloc*
DiskAlloc_New(uint64_t min, uint64_t max, int numaSocket)
{
  NDNDPDK_ASSERT(min > 0);
  NDNDPDK_ASSERT(max >= min);
  // no need for more than 2^32 slots, because PCCT cannot index that many entries
  uint64_t total = RTE_MAX(UINT32_MAX, max - min + 1);
  size_t size = sizeof(DiskAlloc) + total * sizeof(uint32_t);

  DiskAlloc* a = rte_malloc_socket("DiskAlloc", size, 0, numaSocket);
  a->min = min;
  a->count = total;
  a->total = total;
  for (uint32_t i = 0; i < total; ++i) {
    a->arr32[i] = i;
  }
  return a;
}
