#include "cs.h"
#include "pit.h"

void
Cs_SetCapacity(Cs* cs, uint32_t capacity)
{
  assert(false); // not implemented
}

void
Cs_ReplacePitEntry(Cs* cs, PitEntry* pitEntry, struct rte_mbuf* pkt)
{
  CsPriv* csp = Cs_GetPriv(cs);
  PccEntry* pccEntry = __Pit_RawErase(Pcct_GetPit(Pcct_FromCs(cs)), pitEntry);
  pccEntry->hasCsEntry = true;
  ++csp->nEntries;

  CsEntry* entry = &pccEntry->csEntry;
  entry->data = pkt;
}

void
Cs_Erase(Cs* cs, CsEntry* entry)
{
  CsPriv* csp = Cs_GetPriv(cs);
  PccEntry* pccEntry = PccEntry_FromCsEntry(entry);

  rte_pktmbuf_free(entry->data);

  --csp->nEntries;
  Pcct_Erase(Pcct_FromCs(cs), pccEntry);
}
