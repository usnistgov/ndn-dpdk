#ifndef NDNDPDK_DISK_ALLOC_H
#define NDNDPDK_DISK_ALLOC_H

/** @file */

#include "../core/common.h"

/** @brief Simple non-thread-safe disk slot allocator. */
typedef struct DiskAlloc
{
  uint64_t min;
  uint64_t count;
  uint64_t total;
  uint32_t arr32[0];
} DiskAlloc;

/**
 * @brief Allocate a disk slot.
 * @retval 0 no disk slot available.
 */
__attribute__((nonnull)) static __rte_always_inline uint64_t
DiskAlloc_Alloc(DiskAlloc* a)
{
  if (unlikely(a->count == 0)) {
    return 0;
  }
  return a->arr32[--a->count] + a->min;
}

/** @brief Free a disk slot. */
__attribute__((nonnull)) static __rte_always_inline void
DiskAlloc_Free(DiskAlloc* a, uint64_t slot)
{
  a->arr32[a->count++] = slot - a->min;
  NDNDPDK_ASSERT(a->count <= a->total);
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
