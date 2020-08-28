#include "ndt.h"

Ndt*
Ndt_New_(uint64_t nEntries, int numaSocket)
{
  NDNDPDK_ASSERT(rte_is_power_of_2(nEntries));
  size_t sz = sizeof(Ndt) + nEntries * sizeof(((Ndt*)NULL)->table[0]);
  Ndt* ndt = rte_zmalloc_socket("Ndt", sz, RTE_CACHE_LINE_SIZE, numaSocket);
  if (unlikely(ndt == NULL)) {
    abort();
  }

  ndt->indexMask = nEntries - 1;
  for (uint64_t i = 0; i < nEntries; ++i) {
    atomic_init(&ndt->table[i], 0);
  }
  return ndt;
}

NdtThread*
Ndtt_New_(Ndt* ndt, int numaSocket)
{
  size_t sz = sizeof(NdtThread) + (ndt->indexMask + 1) * sizeof(((NdtThread*)NULL)->nHits[0]);
  NdtThread* ndtt = rte_zmalloc_socket("NdtThread", sz, RTE_CACHE_LINE_SIZE, numaSocket);
  if (unlikely(ndtt == NULL)) {
    abort();
  }
  ndtt->ndt = ndt;
  return ndtt;
}
