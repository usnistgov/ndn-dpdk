#include "pit.h"

#include "../../core/logger.h"

INIT_ZF_LOG(Pit);

static void __Pit_Timeout(MinTmr* tmr, void* pit0);

void
Pit_Init(Pit* pit, struct rte_mempool* headerMp, struct rte_mempool* guiderMp,
         struct rte_mempool* indirectMp)
{
  PitPriv* pitp = Pit_GetPriv(pit);

  // 2^12 slots of 33ms interval, accommodates InterestLifetime up to 136533ms
  pitp->timeoutSched =
    MinSched_New(12, rte_get_tsc_hz() / 30, __Pit_Timeout, pit);
  assert(MinSched_GetMaxDelay(pitp->timeoutSched) >=
         PIT_MAX_LIFETIME * rte_get_tsc_hz() / 1000);

  pitp->headerMp = headerMp;
  pitp->guiderMp = guiderMp;
  pitp->indirectMp = indirectMp;
}

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
    assert(false); // XXX not implemented
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
    ZF_LOGD("%p Insert(%s) pcc=%p has-CS cs=%p", pit,
            PccSearch_ToDebugString(&search), pccEntry,
            PccEntry_GetCsEntry(pccEntry));
    return PitInsertResult_New(pccEntry, PIT_INSERT_CS);
  }

  if (interest->mustBeFresh) {
    if (!pccEntry->hasPitEntry1) {
      ++pitp->nEntries;
      pccEntry->hasPitEntry1 = true;
      PitEntry_Init(PccEntry_GetPitEntry1(pccEntry), npkt);
      ZF_LOGD("%p Insert(%s) pcc=%p ins-PIT1 pit=%p", pit,
              PccSearch_ToDebugString(&search), pccEntry,
              PccEntry_GetPitEntry1(pccEntry));
    } else {
      ZF_LOGD("%p Insert(%s) pcc=%p has-PIT1 pit=%p", pit,
              PccSearch_ToDebugString(&search), pccEntry,
              PccEntry_GetPitEntry1(pccEntry));
    }
    return PitInsertResult_New(pccEntry, PIT_INSERT_PIT1);
  }

  if (!pccEntry->hasPitEntry0) {
    assert(!pccEntry->hasCsEntry);
    ++pitp->nEntries;
    pccEntry->hasPitEntry0 = true;
    PitEntry_Init(PccEntry_GetPitEntry0(pccEntry), npkt);
    ZF_LOGD("%p Insert(%s) pcc=%p ins-PIT0 pit=%p", pit,
            PccSearch_ToDebugString(&search), pccEntry,
            PccEntry_GetPitEntry0(pccEntry));
  } else {
    ZF_LOGD("%p Insert(%s) pcc=%p has-PIT0 pit=%p", pit,
            PccSearch_ToDebugString(&search), pccEntry,
            PccEntry_GetPitEntry0(pccEntry));
  }
  return PitInsertResult_New(pccEntry, PIT_INSERT_PIT0);
}

PccEntry*
__Pit_RawErase0(Pit* pit, PitEntry* entry)
{
  assert(entry->mustBeFresh == false);
  PitEntry_Finalize(entry);

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
  PitEntry_Finalize(entry);

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
    ZF_LOGD("%p Erase(%p) del-PIT1 pcc=%p", pit, entry, pccEntry);
    if (pccEntry->hasPitEntry0 || pccEntry->hasCsEntry) {
      return;
    }
  } else {
    pccEntry = __Pit_RawErase0(pit, entry);
    ZF_LOGD("%p Erase(%p) del-PIT0 pcc=%p", pit, entry, pccEntry);
    if (pccEntry->hasPitEntry1) {
      return;
    }
  }
  Pcct_Erase(Pit_ToPcct(pit), pccEntry);
}

static void
__Pit_Timeout(MinTmr* tmr, void* pit0)
{
  Pit* pit = (Pit*)pit0;
  PitEntry* entry = container_of(tmr, PitEntry, timeout);
  ZF_LOGD("%p Timeout(%p)", pit, entry);
  Pit_Erase(pit, entry);
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
