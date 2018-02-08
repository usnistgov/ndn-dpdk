#include "pit.h"

PitInsertResult
Pit_Insert(Pit* pit, const InterestPkt* interest)
{
  PitPriv* pitp = Pit_GetPriv(pit);

  PccKey scratch;
  PccSearch search;
  search.name = Name_Linearize(&interest->name, scratch.name);
  uint64_t hash = Name_ComputeHash(&interest->name);
  if (interest->nFwHints > 0) {
    search.fh = Name_Linearize(&interest->fwHints[0], scratch.fh);
    hash ^= Name_ComputeHash(&interest->fwHints[0]);
  } else {
    search.fh.length = 0;
  }

  bool isNew = false;
  PccEntry* pccEntry = Pcct_Insert(pit, hash, &search, &isNew);
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
  pccEntry->hasPitEntry = false;
  --pitp->nEntries;
  return pccEntry;
}

void
Pit_Erase(Pit* pit, PitEntry* entry)
{
  PccEntry* pccEntry = __Pit_RawErase(pit, entry);
  Pcct_Erase(Pcct_FromPit(pit), pccEntry);
}

PitEntry*
Pit_Find(Pit* pit, uint64_t token)
{
  assert(false); // not implemented
  return NULL;
}
