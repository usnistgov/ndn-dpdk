#include "cs.h"
#include "pit.h"

#include "../core/logger.h"

N_LOG_INIT(Cs);

// Bulk size of CS eviction, also the minimum CS capacity.
#define CS_EVICT_BULK 64

__attribute__((nonnull)) static void
CsEraseBatch_Append_(PcctEraseBatch* peb, CsEntry* entry, const char* isDirectDbg)
{
  PccEntry* pccEntry = PccEntry_FromCsEntry(entry);
  PccEntry_RemoveCsEntry(pccEntry);
  if (likely(!pccEntry->hasEntries)) {
    N_LOGD("^ cs=%p(%s) pcc=%p(erase)", entry, isDirectDbg, pccEntry);
    PcctEraseBatch_Append(peb, pccEntry);
  } else {
    N_LOGD("^ cs=%p(%s) pcc=%p(keep)", entry, isDirectDbg, pccEntry);
  }
}

/** @brief Erase an indirect CS entry. */
__attribute__((nonnull)) static void
CsEraseBatch_AddIndirect(PcctEraseBatch* peb, CsEntry* entry)
{
  NDNDPDK_ASSERT(!CsEntry_IsDirect(entry));
  N_LOGV("^ indirect=%p direct=%p(%" PRId8 ")", entry, entry->direct, entry->direct->nIndirects);
  CsEntry_Finalize(entry);
  CsEraseBatch_Append_(peb, entry, "indirect");
}

/** @brief Erase a direct CS entry; delist and erase indirect entries. */
__attribute__((nonnull)) static void
CsEraseBatch_AddDirect(PcctEraseBatch* peb, CsEntry* entry)
{
  NDNDPDK_ASSERT(CsEntry_IsDirect(entry));
  Cs* cs = &peb->pcct->cs;
  for (int i = 0; i < entry->nIndirects; ++i) {
    CsEntry* indirect = entry->indirect[i];
    CsList_Remove(&cs->indirect, indirect);
    CsEraseBatch_Append_(peb, indirect, "indirect-dep");
  }
  entry->nIndirects = 0;
  CsEntry_Finalize(entry);
  CsEraseBatch_Append_(peb, entry, "direct");
}

/** @brief Erase a CS entry including dependents. */
__attribute__((nonnull)) static void
Cs_Erase_(Cs* cs, CsEntry* entry)
{
  PcctEraseBatch peb = PcctEraseBatch_New(Pcct_FromCs(cs));
  if (CsEntry_IsDirect(entry)) {
    CsArc_Remove(&cs->direct, entry);
    CsEraseBatch_AddDirect(&peb, entry);
  } else {
    CsList_Remove(&cs->indirect, entry);
    CsEraseBatch_AddIndirect(&peb, entry);
  }
  PcctEraseBatch_Finish(&peb);
}

__attribute__((nonnull)) static void
Cs_EvictBulk_(Cs* cs, CsList* csl, const char* cslName, CsList_EvictCb evictCb)
{
  N_LOGD("%p Evict(%s) count=%" PRIu32, cs, cslName, csl->count);
  PcctEraseBatch peb = PcctEraseBatch_New(Pcct_FromCs(cs));
  CsList_EvictBulk(csl, CS_EVICT_BULK, evictCb, &peb);
  PcctEraseBatch_Finish(&peb);
  N_LOGD("^ end-count=%" PRIu32, csl->count);
}

__attribute__((nonnull)) static __rte_always_inline void
Cs_Evict(Cs* cs)
{
  if (unlikely(cs->indirect.count > cs->indirect.capacity)) {
    Cs_EvictBulk_(cs, &cs->indirect, "indirect", (CsList_EvictCb)CsEraseBatch_AddIndirect);
  }
  if (unlikely(cs->direct.Del.count >= CS_EVICT_BULK)) {
    Cs_EvictBulk_(cs, &cs->direct.Del, "direct", (CsList_EvictCb)CsEraseBatch_AddDirect);
  }
}

static CsList*
Cs_GetList_(Cs* cs, CsListID cslId)
{
  switch (cslId) {
    case CslMdT1:
    case CslMdB1:
    case CslMdT2:
    case CslMdB2:
    case CslMdDel:
      return CsArc_GetList(&cs->direct, cslId - CslMd);
    case CslMi:
      return &cs->indirect;
    case CslMd:
    default:
      NDNDPDK_ASSERT(false);
      return NULL;
  }
}

void
Cs_Init(Cs* cs, uint32_t capMd, uint32_t capMi)
{
  capMd = RTE_MAX(capMd, CS_EVICT_BULK);
  capMi = RTE_MAX(capMi, CS_EVICT_BULK);

  CsArc_Init(&cs->direct, capMd);
  CsList_Init(&cs->indirect);
  cs->indirect.capacity = capMi;

  N_LOGI("Init cs=%p arc=%p pcct=%p cap-md=%" PRIu32 " cap-mi=%" PRIu32, cs, &cs->direct,
         Pcct_FromCs(cs), capMd, capMi);
}

uint32_t
Cs_GetCapacity(Cs* cs, CsListID cslId)
{
  if (cslId == CslMd) {
    return CsArc_GetCapacity(&cs->direct);
  }
  return Cs_GetList_(cs, cslId)->capacity;
}

uint32_t
Cs_CountEntries(Cs* cs, CsListID cslId)
{
  if (cslId == CslMd) {
    return CsArc_CountEntries(&cs->direct);
  }
  return Cs_GetList_(cs, cslId)->count;
}

/** @brief Add or refresh a direct entry for @p npkt in @p pccEntry . */
static CsEntry*
Cs_PutDirect(Cs* cs, Packet* npkt, PccEntry* pccEntry)
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  PData* data = Packet_GetDataHdr(npkt);

  CsEntry* entry = NULL;
  if (unlikely(pccEntry->hasCsEntry)) {
    // refresh direct entry
    entry = PccEntry_GetCsEntry(pccEntry);
    N_LOGD("PutDirect(refresh) cs=%p npkt=%p pcc-entry=%p cs-entry=%p", cs, npkt, pccEntry, entry);
    if (CsEntry_IsDirect(entry)) {
      // erase any indirect entry with implicit digest name, because it may not match new Data
      for (int8_t i = 0; i < entry->nIndirects; ++i) {
        CsEntry* indirect = entry->indirect[i];
        PccEntry* indirectPcc = PccEntry_FromCsEntry(indirect);
        if (unlikely(indirectPcc->key.nameL > data->name.length)) {
          N_LOGD("^ erase-implicit-digest-indirect");
          Cs_Erase_(cs, indirect);
          break;
        }
      }
    }
    CsEntry_Clear(entry);
    CsArc_Add(&cs->direct, entry);
  } else {
    // insert direct entry
    entry = PccEntry_AddCsEntry(pccEntry);
    if (unlikely(entry == NULL)) {
      N_LOGW("PutDirect alloc-err cs=%p npkt=%p pcc-entry=%p", cs, npkt, pccEntry);
      return NULL;
    }
    N_LOGD("PutDirect insert cs=%p npkt=%p pcc-entry=%p cs-entry=%p", cs, npkt, pccEntry, entry);
    entry->arcList = 0;
    entry->nIndirects = 0;
    CsArc_Add(&cs->direct, entry);
  }
  entry->data = npkt;
  entry->freshUntil = Mbuf_GetTimestamp(pkt) + TscDuration_FromMillis(data->freshness);
  return entry;
}

/** @brief Insert a direct entry for @p npkt that was retrieved by @p interest . */
static CsEntry*
Cs_InsertDirect(Cs* cs, Packet* npkt, PInterest* interest)
{
  Pcct* pcct = Pcct_FromCs(cs);
  PData* data = Packet_GetDataHdr(npkt);

  // construct PccSearch
  PccSearch search;
  PccSearch_FromNames(&search, &data->name, interest);

  // seek PCC entry
  bool isNewPcc = false;
  PccEntry* pccEntry = Pcct_Insert(pcct, &search, &isNewPcc);
  if (unlikely(pccEntry == NULL)) {
    N_LOGD("InsertDirect alloc-err cs=%p npkt=%p", cs, npkt);
    return NULL;
  }

  // put direct entry on PCC entry
  return Cs_PutDirect(cs, npkt, pccEntry);
}

/** @brief Add or refresh an indirect entry in @p pccEntry and associate with @p direct . */
static bool
Cs_PutIndirect(Cs* cs, CsEntry* direct, PccEntry* pccEntry)
{
  NDNDPDK_ASSERT(!pccEntry->hasPitEntry0);

  CsEntry* entry = NULL;
  if (unlikely(pccEntry->hasCsEntry)) {
    entry = PccEntry_GetCsEntry(pccEntry);
    if (unlikely(CsEntry_IsDirect(entry) && entry->nIndirects > 0)) {
      // don't overwrite direct entry with dependencies
      N_LOGD("PutIndirect has-dependency cs=%p npkt=%p pcc-entry-%p cs-entry=%p", cs, direct,
             pccEntry, entry);
      return false;
    }
    // refresh indirect entry
    // old entry can be either direct without dependency or indirect
    CsEntry_Clear(entry);
    CsList_MoveToLast(&cs->indirect, entry);
    N_LOGD("PutIndirect refresh cs=%p npkt=%p pcc-entry-%p cs-entry=%p count=%" PRIu32, cs, direct,
           pccEntry, entry, cs->indirect.count);
  } else {
    // insert indirect entry
    entry = PccEntry_AddCsEntry(pccEntry);
    if (unlikely(entry == NULL)) {
      N_LOGW("PutIndirect alloc-err cs=%p npkt=%p pcc-entry-%p", cs, direct, pccEntry);
      return NULL;
    }
    entry->nIndirects = 0;
    CsList_Append(&cs->indirect, entry);
    N_LOGD("PutIndirect insert cs=%p npkt=%p pcc-entry-%p cs-entry=%p count=%" PRIu32, cs, direct,
           pccEntry, entry, cs->indirect.count);
  }

  if (likely(CsEntry_Assoc(entry, direct))) {
    N_LOGV("^ indirect=%p direct=%p(%" PRId8 ")", entry, entry->direct, entry->direct->nIndirects);
    return true;
  }

  N_LOGD("^ indirect-assoc-err");
  CsList_Remove(&cs->indirect, entry);
  PccEntry_RemoveCsEntry(pccEntry);
  Pcct_Erase(Pcct_FromCs(cs), pccEntry);
  return false;
}

void
Cs_Insert(Cs* cs, Packet* npkt, PitFindResult pitFound)
{
  Pcct* pcct = Pcct_FromCs(cs);
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  PData* data = Packet_GetDataHdr(npkt);
  PccEntry* pccEntry = pitFound.entry;
  PInterest* interest = PitFindResult_GetInterest(pitFound);
  CsEntry* direct = NULL;

  // if Interest name differs from Data name, insert a direct entry elsewhere
  if (unlikely(interest->name.nComps != data->name.nComps)) {
    direct = Cs_InsertDirect(cs, npkt, interest);
    if (unlikely(direct == NULL)) { // direct entry insertion failed
      Pit_RawErase01_(&pcct->pit, pccEntry);
      rte_pktmbuf_free(pkt);
      if (likely(!pccEntry->hasCsEntry)) {
        Pcct_Erase(pcct, pccEntry);
      }
      return;
    }
    pkt = NULL; // owned by direct entry, don't free it
  }

  // delete PIT entries
  Pit_RawErase01_(&pcct->pit, pccEntry);
  interest = NULL;

  if (likely(direct == NULL)) {
    // put direct CS entry at pccEntry
    direct = Cs_PutDirect(cs, npkt, pccEntry);
    NDNDPDK_ASSERT(direct != NULL);
  } else {
    // put indirect CS entry at pccEntry
    Cs_PutIndirect(cs, direct, pccEntry);
  }

  // evict if over capacity
  Cs_Evict(cs);
}

bool
Cs_MatchInterest(Cs* cs, CsEntry* entry, Packet* interestNpkt)
{
  CsEntry* direct = CsEntry_GetDirect(entry);
  bool hasData = CsEntry_GetData(direct) != NULL;
  PccEntry* pccDirect = PccEntry_FromCsEntry(direct);

  PInterest* interest = Packet_GetInterestHdr(interestNpkt);
  bool violateCanBePrefix = !interest->canBePrefix && interest->name.length < pccDirect->key.nameL;
  bool violateMustBeFresh =
    interest->mustBeFresh &&
    !CsEntry_IsFresh(direct, Mbuf_GetTimestamp(Packet_ToMbuf(interestNpkt)));
  N_LOGD("MatchInterest cs=%p cs-entry=%p~%s cbp=%s mbf=%s has-data=%s", cs, entry,
         CsEntry_IsDirect(entry) ? "direct" : "indirect", violateCanBePrefix ? "N" : "Y",
         violateMustBeFresh ? "N" : "Y", hasData ? "Y" : "N");

  if (likely(!violateCanBePrefix && !violateMustBeFresh)) {
    if (!CsEntry_IsDirect(entry)) {
      CsList_MoveToLast(&cs->indirect, entry);
    }
    if (likely(hasData)) {
      CsArc_Add(&cs->direct, direct);
      return true;
    }
  }
  return false;
}

void
Cs_Erase(Cs* cs, CsEntry* entry)
{
  N_LOGD("Erase cs=%p cs-entry=%p", cs, entry);
  Cs_Erase_(cs, entry);
}
