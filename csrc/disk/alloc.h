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
  uint64_t slab;
  uint32_t pos;
  struct rte_bitmap bmp[0] __rte_cache_aligned;
} __rte_cache_aligned DiskAlloc;
static_assert(offsetof(DiskAlloc, bmp[0]) % RTE_CACHE_LINE_SIZE == 0, "");

/**
 * @brief Allocate a disk slot.
 * @retval 0 no disk slot available.
 */
__attribute__((nonnull)) static inline uint64_t
DiskAlloc_Alloc(DiskAlloc* a)
{
  if (a->slab == 0) {
    int found = rte_bitmap_scan(a->bmp, &a->pos, &a->slab);
    if (unlikely(found == 0)) {
      return 0;
    }
  }

  uint64_t offset = rte_bsf64(a->slab);
  a->slab &= ~((uint64_t)1 << offset);
  uint64_t pos = a->pos + offset;
  rte_bitmap_clear(a->bmp, pos);
  return a->min + pos;
}

/** @brief Free a disk slot. */
__attribute__((nonnull)) static inline void
DiskAlloc_Free(DiskAlloc* a, uint64_t slotID)
{
  NDNDPDK_ASSERT(slotID >= a->min && slotID <= a->max);
  uint32_t pos = slotID - a->min;
  NDNDPDK_ASSERT(rte_bitmap_get(a->bmp, pos) == 0);
  rte_bitmap_set(a->bmp, pos);

  // if s->slab reflects the modified bitmap slab, DiskAlloc_Alloc can't use it right away but
  // will pick up the newly available slotID during bitmap scan
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
