#include "pit.h"

#include "../../core/logger.h"
#include "cs.h"

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

PitInsertResult
Pit_Insert(Pit* pit, Packet* npkt, const FibEntry* fibEntry)
{
  Pcct* pcct = Pit_ToPcct(pit);
  PitPriv* pitp = Pit_GetPriv(pit);
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
  if (pccEntry->hasCsEntry &&
      likely(__Cs_MatchInterest(Cs_FromPcct(pcct), pccEntry, npkt))) {
    // CS entry satisfies Interest
    ZF_LOGD("%p Insert(%s) pcc=%p has-CS", pit,
            PccSearch_ToDebugString(&search), pccEntry);
    ++pitp->nCsMatch;
    return __PitResult_New(pccEntry, PIT_INSERT_CS);
  }

  // add token
  uint64_t token = Pcct_AddToken(pcct, pccEntry);
  if (unlikely(token == 0)) {
    if (isNewPcc) {
      Pcct_Erase(pcct, pccEntry);
    }
    ++pitp->nAllocErr;
    return __PitResult_New(NULL, PIT_INSERT_FULL);
  }

  PitEntry* entry = NULL;
  bool isNew = false;
  PitInsertResultKind resKind = 0;

  // add PIT entry if it does not exist
  if (!interest->mustBeFresh) {
    isNew = !pccEntry->hasPitEntry0;
    entry = PccEntry_AddPitEntry0(pccEntry);
    resKind = PIT_INSERT_PIT0;
  } else {
    isNew = !pccEntry->hasPitEntry1;
    entry = PccEntry_AddPitEntry1(pccEntry);
    resKind = PIT_INSERT_PIT1;
  }

  if (unlikely(entry == NULL)) {
    assert(!isNewPcc); // new PccEntry must have occupied slot1
    ++pitp->nAllocErr;
    return __PitResult_New(NULL, PIT_INSERT_FULL);
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
  PccEntry* pccEntry = PccEntry_FromPitEntry(entry);
  if (!entry->mustBeFresh) {
    assert(pccEntry->hasPitEntry0);
    PccEntry_RemovePitEntry0(pccEntry);
    ZF_LOGD("%p Erase(%p) del-PIT0 pcc=%p", pit, entry, pccEntry);
  } else {
    assert(pccEntry->hasPitEntry1);
    PccEntry_RemovePitEntry1(pccEntry);
    ZF_LOGD("%p Erase(%p) del-PIT1 pcc=%p", pit, entry, pccEntry);
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
    PccEntry_RemovePitEntry0(pccEntry);
  }
  if (pccEntry->hasPitEntry1) {
    --pitp->nEntries;
    PitEntry_Finalize(PccEntry_GetPitEntry1(pccEntry));
    PccEntry_RemovePitEntry1(pccEntry);
  }
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

PitFindResult
Pit_FindByData(Pit* pit, Packet* npkt)
{
  PitPriv* pitp = Pit_GetPriv(pit);
  uint64_t token = Packet_GetLpL3Hdr(npkt)->pitToken;

  PccEntry* pccEntry = Pcct_FindByToken(Pit_ToPcct(pit), token);
  if (unlikely(pccEntry == NULL)) {
    ++pitp->nDataMiss;
    return __PitResult_New(NULL, PIT_FIND_NONE);
  }

  PitFindResultFlag flags = PIT_FIND_NONE;
  PInterest* interest = NULL;
  if (pccEntry->hasPitEntry1) {
    flags |= PIT_FIND_PIT1;
    interest = Packet_GetInterestHdr(PccEntry_GetPitEntry1(pccEntry)->npkt);
  }
  if (pccEntry->hasPitEntry0) {
    flags |= PIT_FIND_PIT0;
    interest = Packet_GetInterestHdr(PccEntry_GetPitEntry0(pccEntry)->npkt);
  }

  if (likely(flags != PIT_FIND_NONE)) {
    PData* data = Packet_GetDataHdr(npkt);
    DataSatisfyResult satisfy = PData_CanSatisfy(data, interest);
    switch (satisfy) {
      case DATA_SATISFY_YES:
        ++pitp->nDataHit;
        break;
      case DATA_SATISFY_NO:
        flags = PIT_FIND_NONE;
        ++pitp->nDataMiss;
        break;
      case DATA_SATISFY_NEED_DIGEST:
        flags |= PIT_FIND_NEED_DIGEST;
        // do not increment either counter: caller should compute Data digest
        // and reinvoke Pit_FindByData that leads to either Data hit or miss.
        break;
    }
  }
  return __PitResult_New(pccEntry, flags);
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
