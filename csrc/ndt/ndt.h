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
  _Atomic uint8_t table[];
} Ndt;

/** @brief Create NDT replica. */
__attribute__((returns_nonnull)) Ndt*
Ndt_New(uint64_t nEntries, int numaSocket);

/** @brief Update an entry. */
__attribute__((nonnull)) static inline void
Ndt_Update(Ndt* ndt, uint64_t index, uint8_t value)
{
  atomic_store_explicit(&ndt->table[index], value, memory_order_relaxed);
}

/** @brief Read entry by index. */
__attribute__((nonnull)) static __rte_always_inline uint8_t
Ndt_Read(Ndt* ndt, uint64_t index)
{
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

/** @brief NDT querier with counters. */
typedef struct NdtQuerier
{
  Ndt* ndt;
  uint64_t nLookups;
  uint32_t nHits[];
} NdtQuerier;

/** @brief Create NDT querier. */
__attribute__((nonnull, returns_nonnull)) NdtQuerier*
NdtQuerier_New(Ndt* ndt, int numaSocket);

/** @brief Query NDT by name with counting. */
__attribute__((nonnull)) static inline uint8_t
NdtQuerier_Lookup(NdtQuerier* ndq, const PName* name)
{
  uint64_t index;
  uint8_t value = Ndt_Lookup(ndq->ndt, name, &index);
  if ((++ndq->nLookups & ndq->ndt->sampleMask) == 0) {
    ++ndq->nHits[index];
  }
  return value;
}

#endif // NDNDPDK_NDT_NDT_H
