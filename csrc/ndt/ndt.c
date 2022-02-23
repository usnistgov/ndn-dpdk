#include "ndt.h"

Ndt*
Ndt_New(uint64_t nEntries, int numaSocket)
{
  NDNDPDK_ASSERT(rte_is_power_of_2(nEntries));
  size_t sz = sizeof(Ndt) + nEntries * RTE_SIZEOF_FIELD(Ndt, table[0]);
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

NdtQuerier*
NdtQuerier_New(Ndt* ndt, int numaSocket)
{
  NdtQuerier* ndq = NULL;
  size_t sz = sizeof(NdtQuerier) + (ndt->indexMask + 1) * sizeof(ndq->nHits[0]);
  ndq = rte_zmalloc_socket("NdtQuerier", sz, RTE_CACHE_LINE_SIZE, numaSocket);
  if (unlikely(ndq == NULL)) {
    abort();
  }
  ndq->ndt = ndt;
  return ndq;
}
