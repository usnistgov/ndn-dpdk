#include "cs.h"
#include "pit.h"

void
Cs_Init(Cs* cs)
{
}

void
Cs_SetCapacity(Cs* cs, uint32_t capacity)
{
  assert(false); // not implemented
}

void
Cs_ReplacePitEntry(Cs* cs, PitEntry* pitEntry, struct Packet* npkt)
{
  CsPriv* csp = Cs_GetPriv(cs);
  // XXX will crash if pitEntry is for MustBeFresh=1
  PccEntry* pccEntry = __Pit_RawErase0(Pit_FromPcct(Cs_ToPcct(cs)), pitEntry);
  pccEntry->hasCsEntry = true;
  ++csp->nEntries;

  CsEntry* entry = &pccEntry->csEntry;
  entry->data = npkt;
}

void
Cs_Erase(Cs* cs, CsEntry* entry)
{
  CsPriv* csp = Cs_GetPriv(cs);
  PccEntry* pccEntry = PccEntry_FromCsEntry(entry);

  rte_pktmbuf_free(Packet_ToMbuf(entry->data));

  --csp->nEntries;
  // XXX incorrect if pitEntry1 presents
  Pcct_Erase(Cs_ToPcct(cs), pccEntry);
}
