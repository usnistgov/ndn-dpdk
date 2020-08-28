#ifndef NDNDPDK_NDT_NDT_H
#define NDNDPDK_NDT_NDT_H

/** @file */

#include "../ndni/name.h"

/** @brief A replica of the Name Dispatch Table (NDT). */
typedef struct Ndt
{
  uint64_t indexMask;
  uint64_t sampleMask;
  uint16_t prefixLen;
  _Atomic uint8_t table[0];
} Ndt;

/** @brief Create NDT replica. */
__attribute__((returns_nonnull)) Ndt*
Ndt_New_(uint64_t nEntries, int numaSocket);

/** @brief Update an entry. */
__attribute__((nonnull)) static inline void
Ndt_Update(Ndt* ndt, uint64_t index, uint8_t value)
{
  NDNDPDK_ASSERT(index == (index & ndt->indexMask));
  atomic_store_explicit(&ndt->table[index], value, memory_order_relaxed);
}

/** @brief Read entry by index. */
__attribute__((nonnull)) static __rte_always_inline uint8_t
Ndt_Read(Ndt* ndt, uint64_t index)
{
  NDNDPDK_ASSERT(index == (index & ndt->indexMask));
  return atomic_load_explicit(&ndt->table[index], memory_order_relaxed);
}

/** @brief Query NDT by name. */
__attribute__((nonnull)) static inline uint8_t
Ndt_Lookup(Ndt* ndt, const PName* name, uint64_t* index)
{
  uint16_t prefixLen = RTE_MIN(name->nComps, ndt->prefixLen);
  LName prefix = PName_GetPrefix(name, prefixLen);
  // compute hash in 'uncached' mode, to reduce workload of FwInput thread
  uint64_t hash = LName_ComputeHash(prefix);
  *index = hash & ndt->indexMask;
  return Ndt_Read(ndt, *index);
}

/** @brief NDT lookup thread with counters. */
typedef struct NdtThread
{
  Ndt* ndt;
  uint64_t nLookups;
  uint32_t nHits[0];
} NdtThread;

/** @brief Create NDT lookup thread. */
__attribute__((nonnull, returns_nonnull)) NdtThread*
Ndtt_New_(Ndt* ndt, int numaSocket);

/** @brief Access array of hit counters. */
__attribute__((nonnull, returns_nonnull)) static inline uint32_t*
Ndtt_Hits_(NdtThread* ndtt)
{
  return ndtt->nHits;
}

/** @brief Query NDT by name with counting. */
__attribute__((nonnull)) static inline uint8_t
Ndtt_Lookup(NdtThread* ndtt, const PName* name)
{
  uint64_t index;
  uint8_t value = Ndt_Lookup(ndtt->ndt, name, &index);
  if ((++ndtt->nLookups & ndtt->ndt->sampleMask) == 0) {
    ++ndtt->nHits[index];
  }
  return value;
}

#endif // NDNDPDK_NDT_NDT_H
