#ifndef NDNDPDK_DISK_ALLOC_H
#define NDNDPDK_DISK_ALLOC_H

/** @file */

#include "../core/common.h"
#include <rte_bitmap.h>

/**
 * @brief Disk slot allocator.
 *
 * This data structure is non-thread-safe.
 */
typedef struct DiskAlloc
{
  uint64_t min;
  uint64_t max;
  struct rte_bitmap bmp[0] __rte_cache_aligned;
} DiskAlloc;

/**
 * @brief Allocate a disk slot.
 * @retval 0 no disk slot available.
 */
__attribute__((nonnull)) static inline uint64_t
DiskAlloc_Alloc(DiskAlloc* a)
{
  uint32_t pos = 0;
  uint64_t slab = 0;
  int found = rte_bitmap_scan(a->bmp, &pos, &slab);
  if (unlikely(found == 0)) {
    return 0;
  }
  pos += rte_bsf64(slab);
  rte_bitmap_clear(a->bmp, pos);
  return a->min + pos;
}

/** @brief Free a disk slot. */
__attribute__((nonnull)) static inline void
DiskAlloc_Free(DiskAlloc* a, uint64_t slot)
{
  NDNDPDK_ASSERT(slot >= a->min && slot <= a->max);
  uint32_t pos = slot - a->min;
  NDNDPDK_ASSERT(rte_bitmap_get(a->bmp, pos) == 0);
  rte_bitmap_set(a->bmp, pos);
}

/**
 * @brief Create DiskAlloc.
 * @param min inclusive minimum disk slot number.
 * @param max inclusive maximum disk slot number.
 * @param numaSocket where to allocate memory.
 */
__attribute__((returns_nonnull)) DiskAlloc*
DiskAlloc_New(uint64_t min, uint64_t max, int numaSocket);

#endif // NDNDPDK_DISK_ALLOC_H
