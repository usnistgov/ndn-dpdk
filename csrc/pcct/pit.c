#include "pit.h"

#include "../core/logger.h"
#include "cs.h"

N_LOG_INIT(Pit);

static void
Pit_SgTimerCb_Empty(__rte_unused Pit* pit, __rte_unused PitEntry* entry,
                    __rte_unused uintptr_t arg) {
  NDNDPDK_ASSERT(false);
}

void
Pit_Init(Pit* pit) {
  // 2^12 slots of 33ms interval, accommodates InterestLifetime up to 136533ms
  pit->timeoutSched = MinSched_New(12, TscHz / 30, PitEntry_Timeout_, (uintptr_t)pit);
  NDNDPDK_ASSERT(MinSched_GetMaxDelay(pit->timeoutSched) >=
                 (TscDuration)(PIT_MAX_LIFETIME * TscHz / 1000));

  pit->sgTimerCb = Pit_SgTimerCb_Empty;
}

void
Pit_SetSgTimerCb(Pit* pit, Pit_SgTimerCb cb, uintptr_t ctx) {
  pit->sgTimerCb = cb;
  pit->sgTimerCtx = ctx;
}

PitInsertResult
Pit_Insert(Pit* pit, Packet* npkt, const FibEntry* fibEntry) {
  Pcct* pcct = Pcct_FromPit(pit);
  PInterest* interest = Packet_GetInterestHdr(npkt);

  // construct PccSearch
  PccSearch search = PccSearch_FromNames(&interest->name, interest);

  // seek PCC entry
  bool isNewPcc = false;
  PccEntry* pccEntry = Pcct_Insert(pcct, &search, &isNewPcc);
  if (unlikely(pccEntry == NULL)) {
    ++pit->nAllocErr;
    return (PitInsertResult){.kind = PIT_INSERT_FULL};
  }

  // check for CS match
  if (pccEntry->hasCsEntry) {
    CsEntry* csEntry = PccEntry_GetCsEntry(pccEntry);
    CsEntry* csDirect = Cs_MatchInterest(&pcct->cs, csEntry, npkt);
    if (likely(csDirect != NULL)) {
      // CS entry satisfies Interest
      N_LOGD("Insert has-CS pit=%p search=%s pcc=%p cs-kind=%s", pit,
             PccSearch_ToDebugString(&search), pccEntry, CsEntryKind_ToString(csDirect->kind));
      return (PitInsertResult){.kind = PIT_INSERT_CS, .csEntry = csDirect};
    }
  }

  // assign token if it does not exist
  uint64_t token = Pcct_AddToken(pcct, pccEntry);
  NDNDPDK_ASSERT(token != 0);

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
    NDNDPDK_ASSERT(!isNewPcc); // can't happen on new PccEntry, whose slot1 is unoccupied
    ++pit->nAllocErr;
    return (PitInsertResult){.kind = PIT_INSERT_FULL};
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

  N_LOGD("^ pcc=%p has-entries=%d", pccEntry, pccEntry->hasEntries);
  return (PitInsertResult){.kind = PIT_INSERT_PIT, .pitEntry = pitEntry};
}

void
Pit_Erase(Pit* pit, PitEntry* entry) {
  PccEntry* pccEntry = entry->pccEntry;
  bool mustBeFresh = entry->mustBeFresh;
  PitEntry_Finalize(entry);
  if (!mustBeFresh) {
    NDNDPDK_ASSERT(pccEntry->hasPitEntry0);
    PccEntry_RemovePitEntry0(pccEntry);
    N_LOGD("Erase del-PIT0 pit=%p pcc-entry=%p pit-entry=%p", pit, pccEntry, entry);
  } else {
    NDNDPDK_ASSERT(pccEntry->hasPitEntry1);
    PccEntry_RemovePitEntry1(pccEntry);
    N_LOGD("Erase del-PIT1 pit=%p pcc-entry=%p pit-entry=%p", pit, pccEntry, entry);
  }
  NULLize(entry);

  --pit->nEntries;
  if (!pccEntry->hasEntries) {
    Pcct_Erase(Pcct_FromPit(pit), pccEntry);
  } else if (!pccEntry->hasPitEntries) {
    Pcct_RemoveToken(Pcct_FromPit(pit), pccEntry);
  }
}

void
Pit_EraseSatisfied(Pit* pit, PitFindResult res) {
  NDNDPDK_ASSERT(!PitFindResult_Is(res, PIT_FIND_NEED_DIGEST));
  if (unlikely(PitFindResult_Is(res, PIT_FIND_NONE))) {
    return;
  }

  PitEntry* entry0 = PitFindResult_GetPitEntry0(res);
  if (entry0 != NULL) {
    --pit->nEntries;
    PitEntry_Finalize(entry0);
    PccEntry_RemovePitEntry0(res.entry);
  }

  PitEntry* entry1 = PitFindResult_GetPitEntry1(res);
  if (entry1 != NULL) {
    --pit->nEntries;
    PitEntry_Finalize(entry1);
    PccEntry_RemovePitEntry1(res.entry);
  }
}

__attribute__((nonnull)) static inline PitFindResultFlag
Pit_MatchData(PitEntry* entry, PData* data, PitFindResultFlag positionFlag) {
  PInterest* interest = Packet_GetInterestHdr(entry->npkt);
  DataSatisfyResult satisfy = PData_CanSatisfy(data, interest);
  switch (satisfy) {
    case DataSatisfyYes:
      return positionFlag;
    case DataSatisfyNo:
      return PIT_FIND_NONE;
    case DataSatisfyNeedDigest:
      return positionFlag | PIT_FIND_NEED_DIGEST;
  }
  NDNDPDK_ASSERT(false);
}

PitFindResult
Pit_FindByData(Pit* pit, Packet* npkt, uint64_t token) {
  PccEntry* pccEntry = Pcct_FindByToken(Pcct_FromPit(pit), token);
  if (unlikely(pccEntry == NULL)) {
    ++pit->nDataMiss;
    return (PitFindResult){.kind = PIT_FIND_NONE};
  }

  PData* data = Packet_GetDataHdr(npkt);
  PitFindResultFlag flags = PIT_FIND_NONE;
  if (pccEntry->hasPitEntry0) {
    flags |= Pit_MatchData(PccEntry_GetPitEntry0(pccEntry), data, PIT_FIND_PIT0);
  }
  if (pccEntry->hasPitEntry1) {
    flags |= Pit_MatchData(PccEntry_GetPitEntry1(pccEntry), data, PIT_FIND_PIT1);
  }

  if (flags == PIT_FIND_NONE) {
    ++pit->nDataMiss;
  } else if ((flags & PIT_FIND_NEED_DIGEST) != 0) {
    // do not increment either counter: caller should compute Data digest
    // and reinvoke Pit_FindByData that leads to either Data hit or miss.
  } else {
    ++pit->nDataHit;
  }
  return (PitFindResult){.entry = pccEntry, .kind = flags};
}

PitEntry*
Pit_FindByNack(Pit* pit, Packet* npkt, uint64_t token) {
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
