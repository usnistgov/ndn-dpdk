#include "ndt.h"
#include <rte_malloc.h>

#define NDT_INDEX_MASK (NDT_MAX_INDEX - 1)

NdtThread**
Ndt_Init(Ndt* ndt, uint16_t prefixLen, uint8_t indexBits, uint8_t sampleFreq,
         uint8_t nThreads, const unsigned* numaSockets)
{
  uint64_t tableSize = 1 << indexBits;
  ndt->prefixLen = prefixLen;
  ndt->indexMask = tableSize - 1;
  ndt->sampleMask = (1 << sampleFreq) - 1;
  ndt->nThreads = nThreads;

  ndt->table = (_Atomic uint8_t*)rte_calloc_socket(
    "NdtTable", tableSize, sizeof(ndt->table[0]), 0, numaSockets[0]);
  ndt->threads = (NdtThread**)rte_malloc_socket(
    "NdtThreads", nThreads * sizeof(ndt->threads[0]), 0, numaSockets[0]);
  for (uint8_t i = 0; i < nThreads; ++i) {
    ndt->threads[i] = (NdtThread*)rte_zmalloc_socket(
      "NdtThread", offsetof(NdtThread, nHits) +
                     tableSize * sizeof(ndt->threads[i]->nHits[0]),
      0, numaSockets[i]);
  }
  return ndt->threads;
}

void
Ndt_Close(Ndt* ndt)
{
  for (uint8_t i = 0; i < ndt->nThreads; ++i) {
    rte_free(ndt->threads[i]);
  }
  rte_free(ndt->threads);
  rte_free(ndt->table);
}

void
Ndt_ReadCounters(Ndt* ndt, uint32_t* cnt)
{
  uint64_t tableSize = ndt->indexMask + 1;
  memset(cnt, 0, tableSize * sizeof(cnt[0]));
  for (uint8_t i = 0; i < ndt->nThreads; ++i) {
    for (uint64_t index = 0; index < tableSize; ++index) {
      cnt[index] += ndt->threads[i]->nHits[index];
    }
  }
}

void
Ndt_Update(Ndt* ndt, uint64_t hash, uint8_t value)
{
  uint64_t index = hash & ndt->indexMask;
  atomic_store_explicit(&ndt->table[index], value, memory_order_relaxed);
}

uint8_t
Ndt_Lookup(const Ndt* ndt, NdtThread* ndtt, const Name* name)
{
  uint16_t prefixLen =
    name->nComps < ndt->prefixLen ? name->nComps : ndt->prefixLen;
  uint64_t hash = Name_ComputePrefixHash(name, prefixLen);
  uint64_t index = hash & ndt->indexMask;

  if ((++ndtt->nLookups & ndt->sampleMask) == 0) {
    ++ndtt->nHits[index];
  }

  return atomic_load_explicit(&ndt->table[index], memory_order_relaxed);
}
