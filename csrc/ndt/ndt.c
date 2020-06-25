#include "ndt.h"

#define NDT_INDEX_MASK (NDT_MAX_INDEX - 1)

NdtThread**
Ndt_Init(Ndt* ndt, uint16_t prefixLen, uint8_t indexBits, uint8_t sampleFreq, uint8_t nThreads,
         const unsigned* sockets)
{
  uint64_t tableSize = 1 << indexBits;
  ndt->prefixLen = prefixLen;
  ndt->indexMask = tableSize - 1;
  ndt->sampleMask = (1 << sampleFreq) - 1;
  ndt->nThreads = nThreads;

  ndt->table = (_Atomic uint8_t*)rte_calloc_socket("NdtTable", tableSize, sizeof(ndt->table[0]), 0,
                                                   sockets[0]);
  ndt->threads =
    (NdtThread**)rte_malloc_socket("NdtThreads", nThreads * sizeof(ndt->threads[0]), 0, sockets[0]);
  for (uint8_t i = 0; i < nThreads; ++i) {
    ndt->threads[i] = (NdtThread*)rte_zmalloc_socket(
      "NdtThread", offsetof(NdtThread, nHits) + tableSize * sizeof(ndt->threads[i]->nHits[0]), 0,
      sockets[i]);
  }
  return ndt->threads;
}

void
Ndt_Update(Ndt* ndt, uint64_t index, uint8_t value)
{
  assert(index == (index & ndt->indexMask));
  atomic_store_explicit(&ndt->table[index], value, memory_order_relaxed);
}
