#include "pit.h"

#include "../core/logger.h"
#include "cs.h"

N_LOG_INIT(Pit);

static void
Pit_SgTimerCb_Empty(Pit* pit, PitEntry* entry, void* arg)
{
  N_LOGD("SgTimerCb pit=%p pit-entry=%p no-callback", pit, entry);
}

void
Pit_Init(Pit* pit)
{
  N_LOGI("Init pit=%p pcct=%p", pit, Pcct_FromPit(pit));

  // 2^12 slots of 33ms interval, accommodates InterestLifetime up to 136533ms
  pit->timeoutSched = MinSched_New(12, rte_get_tsc_hz() / 30, PitEntry_Timeout_, pit);
  NDNDPDK_ASSERT(MinSched_GetMaxDelay(pit->timeoutSched) >=
                 (TscDuration)(PIT_MAX_LIFETIME * rte_get_tsc_hz() / 1000));

  pit->sgTimerCb = Pit_SgTimerCb_Empty;
}

void
Pit_SetSgTimerCb(Pit* pit, Pit_SgTimerCb cb, void* arg)
{
  pit->sgTimerCb = cb;
  pit->sgTimerCbArg = arg;
}

PitInsertResult
Pit_Insert(Pit* pit, Packet* npkt, const FibEntry* fibEntry)
{
  Pcct* pcct = Pcct_FromPit(pit);
  PInterest* interest = Packet_GetInterestHdr(npkt);

  // construct PccSearch
  PccSearch search;
  PccSearch_FromNames(&search, &interest->name, interest);

  // seek PCC entry
  bool isNewPcc = false;
  PccEntry* pccEntry = Pcct_Insert(pcct, &search, &isNewPcc);
  if (unlikely(pccEntry == NULL)) {
    ++pit->nAllocErr;
    return PitResult_New_(NULL, PIT_INSERT_FULL);
  }

  // check for CS match
  if (pccEntry->hasCsEntry && likely(Cs_MatchInterest_(&pcct->cs, pccEntry, npkt))) {
    // CS entry satisfies Interest
    char debugStringBuffer[PccSearchDebugStringLength];
    N_LOGD("Insert has-CS pit=%p search=%s pcc=%p", pit,
           PccSearch_ToDebugString(&search, debugStringBuffer), pccEntry);
    ++pit->nCsMatch;
    return PitResult_New_(pccEntry, PIT_INSERT_CS);
  }

  // add token
  uint64_t token = Pcct_AddToken(pcct, pccEntry);
  if (unlikely(token == 0)) {
    if (isNewPcc) {
      Pcct_Erase(pcct, pccEntry);
    }
    ++pit->nAllocErr;
    return PitResult_New_(NULL, PIT_INSERT_FULL);
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
    NDNDPDK_ASSERT(!isNewPcc); // new PccEntry must have occupied slot1
    ++pit->nAllocErr;
    return PitResult_New_(NULL, PIT_INSERT_FULL);
  }

  // initialize new PIT entry, or refresh FIB entry reference on old PIT entry
  if (isNew) {
    ++pit->nEntries;
    ++pit->nInsert;
    PitEntry_Init(entry, npkt, fibEntry);
    char debugStringBuffer[PccSearchDebugStringLength];
    N_LOGD("Insert ins-PIT%d pit=%p search=%s pcc-entry=%p pit-entry=%p", (int)entry->mustBeFresh,
           pit, PccSearch_ToDebugString(&search, debugStringBuffer), pccEntry, entry);
  } else {
    ++pit->nFound;
    PitEntry_RefreshFibEntry(entry, npkt, fibEntry);
    char debugStringBuffer[PccSearchDebugStringLength];
    N_LOGD("Insert has-PIT%d pit=%p search=%s pcc-entry=%p pit-entry=%p", (int)entry->mustBeFresh,
           pit, PccSearch_ToDebugString(&search, debugStringBuffer), pccEntry, entry);
  }

  return PitResult_New_(pccEntry, resKind);
}

void
Pit_Erase(Pit* pit, PitEntry* entry)
{
  PccEntry* pccEntry = PccEntry_FromPitEntry(entry);
  if (!entry->mustBeFresh) {
    NDNDPDK_ASSERT(pccEntry->hasPitEntry0);
    PccEntry_RemovePitEntry0(pccEntry);
    N_LOGD("Erase del-PIT0 pit=%p pcc-entry=%p pit-entry=%p", pit, pccEntry, entry);
  } else {
    NDNDPDK_ASSERT(pccEntry->hasPitEntry1);
    PccEntry_RemovePitEntry1(pccEntry);
    N_LOGD("Erase del-PIT1 pit=%p pcc-entry=%p pit-entry=%p", pit, pccEntry, entry);
  }
  PitEntry_Finalize(entry);

  --pit->nEntries;
  if (!pccEntry->hasEntries) {
    Pcct_Erase(Pcct_FromPit(pit), pccEntry);
  } else if (!pccEntry->hasPitEntries) {
    Pcct_RemoveToken(Pcct_FromPit(pit), pccEntry);
  }
}

void
Pit_RawErase01_(Pit* pit, PccEntry* pccEntry)
{
  if (pccEntry->hasPitEntry0) {
    --pit->nEntries;
    PitEntry_Finalize(PccEntry_GetPitEntry0(pccEntry));
    PccEntry_RemovePitEntry0(pccEntry);
  }
  if (pccEntry->hasPitEntry1) {
    --pit->nEntries;
    PitEntry_Finalize(PccEntry_GetPitEntry1(pccEntry));
    PccEntry_RemovePitEntry1(pccEntry);
  }
  Pcct_RemoveToken(Pcct_FromPit(pit), pccEntry);
}

PitFindResult
Pit_FindByData(Pit* pit, Packet* npkt, uint64_t token)
{
  PccEntry* pccEntry = Pcct_FindByToken(Pcct_FromPit(pit), token);
  if (unlikely(pccEntry == NULL)) {
    ++pit->nDataMiss;
    return PitResult_New_(NULL, PIT_FIND_NONE);
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
      case DataSatisfyYes:
        ++pit->nDataHit;
        break;
      case DataSatisfyNo:
        flags = PIT_FIND_NONE;
        ++pit->nDataMiss;
        break;
      case DataSatisfyNeedDigest:
        flags |= PIT_FIND_NEED_DIGEST;
        // do not increment either counter: caller should compute Data digest
        // and reinvoke Pit_FindByData that leads to either Data hit or miss.
        break;
    }
  }
  return PitResult_New_(pccEntry, flags);
}

PitEntry*
Pit_FindByNack(Pit* pit, Packet* npkt, uint64_t token)
{
  PNack* nack = Packet_GetNackHdr(npkt);
  PInterest* interest = &nack->interest;

  // find PCC entry by token
  PccEntry* pccEntry = Pcct_FindByToken(Pcct_FromPit(pit), token);
  if (unlikely(pccEntry == NULL)) {
    ++pit->nNackMiss;
    return NULL;
  }

  // find PIT entry
  PitEntry* entry = NULL;
  if (interest->mustBeFresh) {
    if (unlikely(!pccEntry->hasPitEntry1)) {
      ++pit->nNackMiss;
      return NULL;
    }
    entry = PccEntry_GetPitEntry1(pccEntry);
  } else {
    if (unlikely(!pccEntry->hasPitEntry0)) {
      ++pit->nNackMiss;
      return NULL;
    }
    entry = PccEntry_GetPitEntry0(pccEntry);
  }

  // verify Interest name matches PCC key
  const LName* interestName = (const LName*)(&interest->name);
  if (unlikely(!PccKey_MatchName(&pccEntry->key, *interestName))) {
    ++pit->nNackMiss;
    return NULL;
  }

  ++pit->nNackHit;
  return entry;
}
