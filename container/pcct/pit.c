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

PitResult
Pit_Insert(Pit* pit, Packet* npkt)
{
  Pcct* pcct = Pit_ToPcct(pit);
  PitPriv* pitp = Pit_GetPriv(pit);
  PInterest* interest = Packet_GetInterestHdr(npkt);

  // construct PccSearch
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

  // seek PCC entry
  bool isNewPcc = false;
  PccEntry* pccEntry = Pcct_Insert(pcct, hash, &search, &isNewPcc);
  if (unlikely(pccEntry == NULL)) {
    ++pitp->nAllocErr;
    return __PitResult_New(NULL, PIT_INSERT_FULL);
  }

  // check for CS match
  if (pccEntry->hasCsEntry) {
    bool isCsMatch = true;
    // TODO CS should not match if it violates MustBeFresh
    // TODO CS should not match if it violates CanBePrefix
    // TODO evict CS entry if it violates CanBePrefix and Interest has MustBeFresh=0,
    //      to make room for pitEntry0
    ZF_LOGD("%p Insert(%s) pcc=%p has-CS cs=%p", pit,
            PccSearch_ToDebugString(&search), pccEntry,
            PccEntry_GetCsEntry(pccEntry));
    ++pitp->nCsMatch;
    return __PitResult_New(pccEntry, PIT_INSERT_CS);
  }

  // add token now, to avoid token allocation error later
  uint64_t token = Pcct_AddToken(pcct, pccEntry);
  if (unlikely(token == 0)) {
    if (isNewPcc) {
      Pcct_Erase(pcct, pccEntry);
    }
    return __PitResult_New(pccEntry, PIT_INSERT_FULL);
  }

  // put PIT entry in slot 1 if MustBeFresh=1
  if (interest->mustBeFresh) {
    if (!pccEntry->hasPitEntry1) {
      ++pitp->nEntries;
      pccEntry->hasPitEntry1 = true;
      PitEntry_Init(PccEntry_GetPitEntry1(pccEntry), npkt);
      ZF_LOGD("%p Insert(%s) pcc=%p ins-PIT1 pit=%p", pit,
              PccSearch_ToDebugString(&search), pccEntry,
              PccEntry_GetPitEntry1(pccEntry));
      ++pitp->nInsert;
    } else {
      ZF_LOGD("%p Insert(%s) pcc=%p has-PIT1 pit=%p", pit,
              PccSearch_ToDebugString(&search), pccEntry,
              PccEntry_GetPitEntry1(pccEntry));
      ++pitp->nFound;
    }
    return __PitResult_New(pccEntry, PIT_INSERT_PIT1);
  }

  // put PIT entry in slot 0 if MustBeFresh=0
  if (!pccEntry->hasPitEntry0) {
    assert(!pccEntry->hasCsEntry);
    ++pitp->nEntries;
    pccEntry->hasPitEntry0 = true;
    PitEntry_Init(PccEntry_GetPitEntry0(pccEntry), npkt);
    ZF_LOGD("%p Insert(%s) pcc=%p ins-PIT0 pit=%p", pit,
            PccSearch_ToDebugString(&search), pccEntry,
            PccEntry_GetPitEntry0(pccEntry));
    ++pitp->nInsert;
  } else {
    ZF_LOGD("%p Insert(%s) pcc=%p has-PIT0 pit=%p", pit,
            PccSearch_ToDebugString(&search), pccEntry,
            PccEntry_GetPitEntry0(pccEntry));
    ++pitp->nFound;
  }
  return __PitResult_New(pccEntry, PIT_INSERT_PIT0);
}

void
Pit_Erase(Pit* pit, PitEntry* entry)
{
  PccEntry* pccEntry;
  if (entry->mustBeFresh) {
    pccEntry = PccEntry_FromPitEntry1(entry);
    assert(pccEntry->hasPitEntry1);
    pccEntry->hasPitEntry1 = false;
    ZF_LOGD("%p Erase(%p) del-PIT1 pcc=%p", pit, entry, pccEntry);
  } else {
    pccEntry = PccEntry_FromPitEntry0(entry);
    assert(pccEntry->hasPitEntry0);
    pccEntry->hasPitEntry0 = false;
    ZF_LOGD("%p Erase(%p) del-PIT0 pcc=%p", pit, entry, pccEntry);
  }
  PitEntry_Finalize(entry);

  PitPriv* pitp = Pit_GetPriv(pit);
  --pitp->nEntries;
  if (!pccEntry->hasEntries) {
    Pcct_Erase(Pit_ToPcct(pit), pccEntry);
  } else if (!pccEntry->hasPitEntries) {
    Pcct_RemoveToken(Pit_ToPcct(pit), pccEntry);
  }
}

void
__Pit_RawErase01(Pit* pit, PccEntry* pccEntry)
{
  PitPriv* pitp = Pit_GetPriv(pit);
  if (pccEntry->hasPitEntry0) {
    --pitp->nEntries;
    PitEntry_Finalize(PccEntry_GetPitEntry0(pccEntry));
  }
  if (pccEntry->hasPitEntry1) {
    --pitp->nEntries;
    PitEntry_Finalize(PccEntry_GetPitEntry1(pccEntry));
  }
  pccEntry->hasPitEntries = 0;
  Pcct_RemoveToken(Pit_ToPcct(pit), pccEntry);
}

static void
__Pit_Timeout(MinTmr* tmr, void* pit0)
{
  Pit* pit = (Pit*)pit0;
  PitEntry* entry = container_of(tmr, PitEntry, timeout);
  ZF_LOGD("%p Timeout(%p)", pit, entry);
  Pit_Erase(pit, entry);
}

PitResult
Pit_FindByData(Pit* pit, Packet* npkt)
{
  PitPriv* pitp = Pit_GetPriv(pit);
  uint64_t token = Packet_GetLpL3Hdr(npkt)->pitToken;

  PccEntry* pccEntry = Pcct_FindByToken(Pit_ToPcct(pit), token);
  if (unlikely(pccEntry == NULL)) {
    ++pitp->nMisses;
    return __PitResult_New(NULL, PIT_FIND_NONE);
  }

  PitResultKind resKind = __PitFindResult_DetermineKind(pccEntry);
  if (likely(resKind != PIT_FIND_NONE)) {
    PInterest* interest = __PitFindResult_GetInterest2(pccEntry, resKind);
    if (unlikely(!PInterest_MatchesData(interest, npkt))) {
      // Data carries old/bad PIT token
      ++pitp->nMisses;
      return __PitResult_New(NULL, PIT_FIND_NONE);
    }
  }

  ++pitp->nHits;
  return __PitResult_New(pccEntry, resKind);
}
