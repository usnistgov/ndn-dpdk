#include "pit.h"

#include "../../core/logger.h"

INIT_ZF_LOG(Pit);

static PitInsertResult
PitInsertResult_New(PccEntry* pccEntry, PitInsertResultKind kind)
{
  PitInsertResult res = {.ptr = ((uintptr_t)pccEntry | kind) };
  assert((res.ptr & ~__PIT_INSERT_MASK) == (uintptr_t)pccEntry);
  return res;
}

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
  if (unlikely(pccEntry == NULL)) {
    return PitInsertResult_New(pccEntry, PIT_INSERT_FULL);
  }

  if (pccEntry->hasCsEntry) {
    bool isCsMatch = true;
    // TODO CS should not match if it violates MustBeFresh
    // TODO CS should not match if it violates CanBePrefix
    // TODO evict CS entry if it violates CanBePrefix and Interest has MustBeFresh=0,
    //      to make room for pitEntry0
    ZF_LOGD("%p Insert(%s) %p has-CS", pit, PccSearch_ToDebugString(&search),
            pccEntry);
    return PitInsertResult_New(pccEntry, PIT_INSERT_CS);
  }

  if (interest->mustBeFresh) {
    if (!pccEntry->hasPitEntry1) {
      ++pitp->nEntries;
      pccEntry->hasPitEntry1 = true;
      pccEntry->pitEntry1.mustBeFresh = true;
      ZF_LOGD("%p Insert(%s) %p ins-PIT1", pit,
              PccSearch_ToDebugString(&search), pccEntry);
    } else {
      ZF_LOGD("%p Insert(%s) %p has-PIT1", pit,
              PccSearch_ToDebugString(&search), pccEntry);
    }
    return PitInsertResult_New(pccEntry, PIT_INSERT_PIT1);
  }

  if (!pccEntry->hasPitEntry0) {
    assert(!pccEntry->hasCsEntry);
    ++pitp->nEntries;
    pccEntry->hasPitEntry0 = true;
    pccEntry->pitEntry0.mustBeFresh = false;
    ZF_LOGD("%p Insert(%s) %p ins-PIT0", pit, PccSearch_ToDebugString(&search),
            pccEntry);
  } else {
    ZF_LOGD("%p Insert(%s) %p has-PIT0", pit, PccSearch_ToDebugString(&search),
            pccEntry);
  }
  return PitInsertResult_New(pccEntry, PIT_INSERT_PIT0);
}

PccEntry*
__Pit_RawErase0(Pit* pit, PitEntry* entry)
{
  assert(entry->mustBeFresh == false);

  PitPriv* pitp = Pit_GetPriv(pit);
  PccEntry* pccEntry = PccEntry_FromPitEntry0(entry);
  --pitp->nEntries;
  pccEntry->hasPitEntry0 = false;

  if (!pccEntry->hasPitEntry1) {
    Pcct_RemoveToken(Pit_ToPcct(pit), pccEntry);
  }
  return pccEntry;
}

PccEntry*
__Pit_RawErase1(Pit* pit, PitEntry* entry)
{
  assert(entry->mustBeFresh == true);

  PitPriv* pitp = Pit_GetPriv(pit);
  PccEntry* pccEntry = PccEntry_FromPitEntry1(entry);
  --pitp->nEntries;
  pccEntry->hasPitEntry1 = false;

  if (!pccEntry->hasPitEntry0) {
    Pcct_RemoveToken(Pit_ToPcct(pit), pccEntry);
  }
  return pccEntry;
}

void
Pit_Erase(Pit* pit, PitEntry* entry)
{
  PccEntry* pccEntry;
  if (entry->mustBeFresh) {
    pccEntry = __Pit_RawErase1(pit, entry);
    ZF_LOGD("%p Erase(%p) del-PIT1", pit, pccEntry);
    if (pccEntry->hasPitEntry0 || pccEntry->hasCsEntry) {
      return;
    }
  } else {
    pccEntry = __Pit_RawErase0(pit, entry);
    ZF_LOGD("%p Erase(%p) del-PIT0", pit, pccEntry);
    if (pccEntry->hasPitEntry1) {
      return;
    }
  }
  Pcct_Erase(Pit_ToPcct(pit), pccEntry);
}

PitFindResult
Pit_Find(Pit* pit, uint64_t token)
{
  PitFindResult res;
  int nMatches = 0;

  PccEntry* pccEntry = Pcct_FindByToken(Pit_ToPcct(pit), token);
  if (likely(pccEntry != NULL)) {
    if (pccEntry->hasPitEntry0) {
      res.matches[nMatches++] = &pccEntry->pitEntry0;
    }
    if (pccEntry->hasPitEntry1) {
      res.matches[nMatches++] = &pccEntry->pitEntry1;
    }
  }

  res.matches[nMatches] = NULL;
  return res;
}
