#include "alloc.h"

DiskAlloc*
DiskAlloc_New(uint64_t min, uint64_t max, int numaSocket) {
  NDNDPDK_ASSERT(min > 0);
  NDNDPDK_ASSERT(max >= min);
  // no need for more than 2^32 slots, because PCCT cannot index that many entries
  uint32_t nSlots = RTE_MIN(UINT32_MAX, max - min + 1);
  uint32_t bmpSize = rte_bitmap_get_memory_footprint(nSlots);

  DiskAlloc* a =
    rte_malloc_socket("DiskAlloc", sizeof(DiskAlloc) + bmpSize, RTE_CACHE_LINE_SIZE, numaSocket);
  a->min = min;
  a->max = min + nSlots - 1;
  struct rte_bitmap* bmp = rte_bitmap_init_with_all_set(nSlots, (uint8_t*)a->bmp, bmpSize);
  NDNDPDK_ASSERT(bmp == a->bmp);
  return a;
}
