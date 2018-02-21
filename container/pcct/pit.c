#include "pit.h"

PitInsertResult
Pit_Insert(Pit* pit, Packet* npkt)
{
  PitPriv* pitp = Pit_GetPriv(pit);
  PInterest* interest = Packet_GetInterestHdr(npkt);

  PccSearch search;
  search.name = *(const LName*)(&interest->name);
  uint64_t hash = PName_ComputeHash(&interest->name.p, interest->name.v);
  if (interest->nFhs > 0) {
    assert(false); // not implemented
    // search.fh = Name_Linearize(&interest->fwHints[0], scratch.fh);
    // hash ^= Name_ComputeHash(&interest->fwHints[0]);
  } else {
    search.fh.length = 0;
  }

  bool isNew = false;
  PccEntry* pccEntry = Pcct_Insert(Pit_ToPcct(pit), hash, &search, &isNew);
  if (likely(pccEntry != NULL) && isNew) {
    pccEntry->hasPitEntry = true;
    ++pitp->nEntries;
  }
  return pccEntry;
}

PccEntry*
__Pit_RawErase(Pit* pit, PitEntry* entry)
{
  PitPriv* pitp = Pit_GetPriv(pit);
  PccEntry* pccEntry = PccEntry_FromPitEntry(entry);
  Pcct_RemoveToken(Pit_ToPcct(pit), pccEntry);
  pccEntry->hasPitEntry = false;
  --pitp->nEntries;
  return pccEntry;
}
