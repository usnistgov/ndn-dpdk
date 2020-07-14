#ifndef NDNDPDK_NDT_NDT_H
#define NDNDPDK_NDT_NDT_H

/** @file */

#include "../ndni/name.h"

/** @brief Per-thread counters for NDT. */
typedef struct NdtThread
{
  uint64_t nLookups;
  uint16_t nHits[0];
} NdtThread;

/** @brief The Name Dispatch Table (NDT). */
typedef struct Ndt
{
  _Atomic uint8_t* table;
  uint64_t indexMask;
  uint64_t sampleMask;
  uint16_t prefixLen;
  uint8_t nThreads;
  NdtThread* threads[RTE_MAX_LCORE];
} Ndt;

/**
 * @brief Update NDT record.
 * @param index table index.
 * @param value new PIT partition number in the table entry.
 */
__attribute__((nonnull)) static inline void
Ndt_Update(Ndt* ndt, uint64_t index, uint8_t value)
{
  NDNDPDK_ASSERT(index == (index & ndt->indexMask));
  atomic_store_explicit(&ndt->table[index], value, memory_order_relaxed);
}

/** @brief Read NDT record. */
__attribute__((nonnull)) static __rte_always_inline uint8_t
Ndt_Read(const Ndt* ndt, uint64_t index)
{
  NDNDPDK_ASSERT(index == (index & ndt->indexMask));
  return atomic_load_explicit(&ndt->table[index], memory_order_relaxed);
}

/** @brief Query NDT without counting. */
__attribute__((nonnull)) static inline uint8_t
Ndt_Lookup(const Ndt* ndt, const PName* name, uint64_t* index)
{
  uint16_t prefixLen = RTE_MIN(name->nComps, ndt->prefixLen);
  LName prefix = PName_GetPrefix(name, prefixLen);
  uint64_t hash = LName_ComputeHash(prefix);
  // Compute hash in 'uncached' mode, to reduce workload of FwInput thread
  *index = hash & ndt->indexMask;
  return Ndt_Read(ndt, *index);
}

/** @brief Query NDT with counting. */
__attribute__((nonnull)) static inline uint8_t
Ndtt_Lookup(const Ndt* ndt, NdtThread* ndtt, const PName* name)
{
  uint64_t index;
  uint8_t value = Ndt_Lookup(ndt, name, &index);
  if ((++ndtt->nLookups & ndt->sampleMask) == 0) {
    ++ndtt->nHits[index];
  }
  return value;
}

#endif // NDNDPDK_NDT_NDT_H
