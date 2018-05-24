#include "pit.h"

#include "../../core/logger.h"

INIT_ZF_LOG(Pit);

static void __Pit_Timeout(MinTmr* tmr, void* pit0);

void
Pit_Init(Pit* pit)
{
  PitPriv* pitp = Pit_GetPriv(pit);
  ZF_LOGI("%p Init() priv=%p", pit, pitp);

  // 2^12 slots of 33ms interval, accommodates InterestLifetime up to 136533ms
  pitp->timeoutSched =
    MinSched_New(12, rte_get_tsc_hz() / 30, __Pit_Timeout, pit);
  assert(MinSched_GetMaxDelay(pitp->timeoutSched) >=
         PIT_MAX_LIFETIME * rte_get_tsc_hz() / 1000);
}

PitResult
Pit_Insert(Pit* pit, Packet* npkt, const FibEntry* fibEntry)
{
  Pcct* pcct = Pit_ToPcct(pit);
  PitPriv* pitp = Pit_GetPriv(pit);
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  PInterest* interest = Packet_GetInterestHdr(npkt);

  // construct PccSearch
  PccSearch search = { 0 };
  search.name = *(const LName*)(&interest->name);
  search.nameHash = PName_ComputeHash(&interest->name.p, interest->name.v);
  if (interest->activeFh >= 0) {
    search.fh = *(const LName*)(&interest->activeFhName);
    search.fhHash =
      PName_ComputeHash(&interest->activeFhName.p, interest->activeFhName.v);
  }

  // seek PCC entry
  bool isNewPcc = false;
  PccEntry* pccEntry = Pcct_Insert(pcct, &search, &isNewPcc);
  if (unlikely(pccEntry == NULL)) {
    ++pitp->nAllocErr;
    return __PitResult_New(NULL, PIT_INSERT_FULL);
  }

  // check for CS match
  if (pccEntry->hasCsEntry) {
    CsEntry* csEntry = PccEntry_GetCsEntry(pccEntry);
    bool isCsMatch =
      !interest->mustBeFresh || CsEntry_IsFresh(csEntry, pkt->timestamp);
    // TODO CS should not match if it violates CanBePrefix
    // TODO evict CS entry if it violates CanBePrefix and Interest has MustBeFresh=0,
    //      to make room for pitEntry0
    if (isCsMatch) {
      ZF_LOGD("%p Insert(%s) pcc=%p has-CS cs=%p", pit,
              PccSearch_ToDebugString(&search), pccEntry, csEntry);
      ++pitp->nCsMatch;
      return __PitResult_New(pccEntry, PIT_INSERT_CS);
    }
  }

  // add token now, to avoid token allocation error later
  uint64_t token = Pcct_AddToken(pcct, pccEntry);
  if (unlikely(token == 0)) {
    if (isNewPcc) {
      Pcct_Erase(pcct, pccEntry);
    }
    return __PitResult_New(pccEntry, PIT_INSERT_FULL);
  }

  PitEntry* entry = NULL;
  bool isNew = false;
  PitResultKind resKind = 0;

  // select slot 0 or 1 according to MustBeFresh
  if (!interest->mustBeFresh) {
    if (!pccEntry->hasPitEntry0) {
      assert(!pccEntry->hasCsEntry);
      pccEntry->hasPitEntry0 = true;
      isNew = true;
    }
    entry = PccEntry_GetPitEntry0(pccEntry);
    resKind = PIT_INSERT_PIT0;
  } else {
    if (!pccEntry->hasPitEntry1) {
      pccEntry->hasPitEntry1 = true;
      isNew = true;
    }
    entry = PccEntry_GetPitEntry1(pccEntry);
    resKind = PIT_INSERT_PIT1;
  }

  // initialize new PIT entry, or refresh FIB entry reference on old PIT entry
  if (isNew) {
    ++pitp->nEntries;
    ++pitp->nInsert;
    PitEntry_Init(entry, npkt, fibEntry);
    ZF_LOGD("%p Insert(%s) pcc=%p ins-PIT%d pit=%p", pit,
            PccSearch_ToDebugString(&search), pccEntry, (int)entry->mustBeFresh,
            entry);
  } else {
    ++pitp->nFound;
    PitEntry_RefreshFibEntry(entry, npkt, fibEntry);
    ZF_LOGD("%p Insert(%s) pcc=%p has-PIT%d pit=%p", pit,
            PccSearch_ToDebugString(&search), pccEntry, (int)entry->mustBeFresh,
            entry);
  }

  return __PitResult_New(pccEntry, resKind);
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
    ++pitp->nDataMiss;
    return __PitResult_New(NULL, PIT_FIND_NONE);
  }

  PitResultKind resKind = __PitFindResult_DetermineKind(pccEntry);
  if (likely(resKind != PIT_FIND_NONE)) {
    PInterest* interest = __PitFindResult_GetInterest2(pccEntry, resKind);
    if (unlikely(!PInterest_MatchesData(interest, npkt))) {
      // Data carries old/bad PIT token
      ++pitp->nDataMiss;
      return __PitResult_New(NULL, PIT_FIND_NONE);
    }
  }

  ++pitp->nDataHit;
  return __PitResult_New(pccEntry, resKind);
}

PitEntry*
Pit_FindByNack(Pit* pit, Packet* npkt)
{
  PitPriv* pitp = Pit_GetPriv(pit);
  uint64_t token = Packet_GetLpL3Hdr(npkt)->pitToken;
  PNack* nack = Packet_GetNackHdr(npkt);
  PInterest* interest = &nack->interest;

  // find PCC entry by token
  PccEntry* pccEntry = Pcct_FindByToken(Pit_ToPcct(pit), token);
  if (unlikely(pccEntry == NULL)) {
    ++pitp->nNackMiss;
    return NULL;
  }

  // find PIT entry
  PitEntry* entry = NULL;
  if (interest->mustBeFresh) {
    if (unlikely(!pccEntry->hasPitEntry1)) {
      ++pitp->nNackMiss;
      return NULL;
    }
    entry = PccEntry_GetPitEntry1(pccEntry);
  } else {
    if (unlikely(!pccEntry->hasPitEntry0)) {
      ++pitp->nNackMiss;
      return NULL;
    }
    entry = PccEntry_GetPitEntry0(pccEntry);
  }

  // verify Interest name matches PCC key
  LName interestName = *(const LName*)(&interest->name);
  if (unlikely(!PccKey_MatchName(&pccEntry->key, interestName))) {
    ++pitp->nNackMiss;
    return NULL;
  }

  ++pitp->nNackHit;
  return entry;
}
