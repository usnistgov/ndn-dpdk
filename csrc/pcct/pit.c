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
  pit->timeoutSched = MinSched_New(12, TscHz / 30, PitEntry_Timeout_, (uintptr_t)pit);
  NDNDPDK_ASSERT(MinSched_GetMaxDelay(pit->timeoutSched) >=
                 (TscDuration)(PIT_MAX_LIFETIME * TscHz / 1000));

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
  PccSearch search = PccSearch_FromNames(&interest->name, interest);

  // seek PCC entry
  bool isNewPcc = false;
  PccEntry* pccEntry = Pcct_Insert(pcct, &search, &isNewPcc);
  if (unlikely(pccEntry == NULL)) {
    ++pit->nAllocErr;
    return (PitInsertResult){ .kind = PIT_INSERT_FULL };
  }

  // check for CS match
  if (pccEntry->hasCsEntry) {
    CsEntry* csEntry = PccEntry_GetCsEntry(pccEntry);
    if (likely(Cs_MatchInterest(&pcct->cs, csEntry, npkt))) {
      // CS entry satisfies Interest
      N_LOGD("Insert has-CS pit=%p search=%s pcc=%p", pit, PccSearch_ToDebugString(&search),
             pccEntry);
      return (PitInsertResult){ .kind = PIT_INSERT_CS, .csEntry = CsEntry_GetDirect(csEntry) };
    }
  }

  // assign token if it does not exist
  uint64_t token = Pcct_AddToken(pcct, pccEntry);
  if (unlikely(token == 0)) {
    if (isNewPcc) {
      Pcct_Erase(pcct, pccEntry);
    }
    ++pit->nAllocErr;
    return (PitInsertResult){ .kind = PIT_INSERT_FULL };
  }

  PitEntry* pitEntry = NULL;
  bool isNew = false;

  // add PIT entry if it does not exist
  if (!interest->mustBeFresh) {
    isNew = !pccEntry->hasPitEntry0;
    pitEntry = PccEntry_AddPitEntry0(pccEntry);
  } else {
    isNew = !pccEntry->hasPitEntry1;
    pitEntry = PccEntry_AddPitEntry1(pccEntry);
  }

  if (unlikely(pitEntry == NULL)) {
    NDNDPDK_ASSERT(!isNewPcc); // new PccEntry should have unoccupied slot1
    ++pit->nAllocErr;
    return (PitInsertResult){ .kind = PIT_INSERT_FULL };
  }

  // initialize new PIT entry, or refresh FIB entry reference on old PIT entry
  if (isNew) {
    ++pit->nEntries;
    ++pit->nInsert;
    PitEntry_Init(pitEntry, npkt, fibEntry);
    N_LOGD("Insert ins-PIT%d pit=%p search=%s pcc-entry=%p pit-entry=%p",
           (int)pitEntry->mustBeFresh, pit, PccSearch_ToDebugString(&search), pccEntry, pitEntry);
  } else {
    ++pit->nFound;
    PitEntry_RefreshFibEntry(pitEntry, npkt, fibEntry);
    N_LOGD("Insert has-PIT%d pit=%p search=%s pcc-entry=%p pit-entry=%p",
           (int)pitEntry->mustBeFresh, pit, PccSearch_ToDebugString(&search), pccEntry, pitEntry);
  }

  return (PitInsertResult){ .kind = PIT_INSERT_PIT, .pitEntry = pitEntry };
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
    return (PitFindResult){ .kind = PIT_FIND_NONE };
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
  return (PitFindResult){ .entry = pccEntry, .kind = flags };
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
  if (unlikely(!PccKey_MatchName(&pccEntry->key, PName_ToLName(&interest->name)))) {
    ++pit->nNackMiss;
    return NULL;
  }

  ++pit->nNackHit;
  return entry;
}
