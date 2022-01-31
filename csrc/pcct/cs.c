#include "cs.h"
#include "cs-disk.h"
#include "pit.h"

#include "../core/logger.h"

N_LOG_INIT(Cs);

__attribute__((nonnull)) static void
CsEraseBatch_Append(PcctEraseBatch* peb, CsEntry* entry, const char* isDirectDbg)
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
CsEraseBatch_AddIndirect(void* peb0, CsEntry* entry)
{
  PcctEraseBatch* peb = (PcctEraseBatch*)peb0;
  N_LOGV("^ indirect=%p direct=%p(%" PRId8 ")", entry, entry->direct, entry->direct->nIndirects);

  CsEntry_Finalize(entry);
  CsEraseBatch_Append(peb, entry, "indirect");
}

/** @brief Erase a direct CS entry; delist and erase indirect entries. */
__attribute__((nonnull)) static void
CsEraseBatch_AddDirect(void* peb0, CsEntry* entry)
{
  PcctEraseBatch* peb = (PcctEraseBatch*)peb0;
  Cs* cs = &peb->pcct->cs;
  for (int i = 0; i < entry->nIndirects; ++i) {
    CsEntry* indirect = entry->indirect[i];
    CsList_Remove(&cs->indirect, indirect);
    CsEraseBatch_Append(peb, indirect, "indirect-dep");
  }
  entry->nIndirects = 0;

  CsEntry_Finalize(entry);
  CsEraseBatch_Append(peb, entry, "direct");
}

/** @brief Erase a CS entry including dependents. */
__attribute__((nonnull)) static void
Cs_EraseEntry(Cs* cs, CsEntry* entry)
{
  PcctEraseBatch peb = PcctEraseBatch_New(Pcct_FromCs(cs));
  switch (entry->kind) {
    case CsEntryIndirect:
      CsList_Remove(&cs->indirect, entry);
      CsEraseBatch_AddIndirect(&peb, entry);
      break;
    case CsEntryDisk:
      CsDisk_Delete(cs, entry);
      // fallthrough
    case CsEntryNone:
    case CsEntryMemory:
      CsArc_Remove(&cs->direct, entry);
      CsEraseBatch_AddDirect(&peb, entry);
      break;
  }
  PcctEraseBatch_Finish(&peb);
}

__attribute__((nonnull)) static void
Cs_Evict(Cs* cs, CsList* csl, const char* cslName, CsList_EvictCb evictCb)
{
  N_LOGD("%p Evict(%s) count=%" PRIu32, cs, cslName, csl->count);
  PcctEraseBatch peb = PcctEraseBatch_New(Pcct_FromCs(cs));
  CsList_EvictBulk(csl, CsEvictBulk, evictCb, &peb);
  PcctEraseBatch_Finish(&peb);
  N_LOGD("^ end-count=%" PRIu32, csl->count);
}

__attribute__((nonnull, returns_nonnull)) static inline CsList*
Cs_GetList(Cs* cs, CsListID l)
{
  if (l == CslIndirect) {
    return &cs->indirect;
  }
  return CsArc_GetList(&cs->direct, l);
}

void
Cs_Init(Cs* cs, uint32_t capDirect, uint32_t capIndirect)
{
  CsArc_Init(&cs->direct, capDirect);
  CsList_Init(&cs->indirect);
  cs->indirect.capacity = capIndirect;

  N_LOGI("Init cs=%p arc=%p pcct=%p cap-direct=%" PRIu32 " cap-indirect=%" PRIu32, cs, &cs->direct,
         Pcct_FromCs(cs), capDirect, capIndirect);
}

uint32_t
Cs_GetCapacity(Cs* cs, CsListID l)
{
  if (l == CslDirect) {
    return CsArc_GetCapacity(&cs->direct);
  }
  return Cs_GetList(cs, l)->capacity;
}

uint32_t
Cs_CountEntries(Cs* cs, CsListID l)
{
  if (l == CslDirect) {
    return CsArc_CountEntries(&cs->direct);
  }
  return Cs_GetList(cs, l)->count;
}

/**
 * @brief Erase indirect entry with implicit digest.
 *
 * This is called before refreshing a direct entry with newly received Data.
 * It is necessary because the new Data could have the same name but different implicit digest,
 * so that the existing implicit digest indirect entry would no longer match the Data.
 */
__attribute__((nonnull)) static inline void
Cs_EraseImplicitDigestIndirect(Cs* cs, CsEntry* direct, size_t dataNameL)
{
  for (uint8_t i = 0; i < direct->nIndirects; ++i) {
    CsEntry* indirect = direct->indirect[i];
    PccEntry* indirectPcc = PccEntry_FromCsEntry(indirect);
    if (unlikely(indirectPcc->key.nameL > dataNameL)) {
      N_LOGD("^ erase-implicit-digest-indirect indirect=%p direct=%p", indirect, direct);
      Cs_EraseEntry(cs, indirect);
      break;
    }
  }
}

/** @brief Add or refresh a direct entry for @p npkt in @p pccEntry . */
__attribute__((nonnull)) static CsEntry*
Cs_PutDirect(Cs* cs, Packet* npkt, PccEntry* pccEntry)
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  PData* data = Packet_GetDataHdr(npkt);

  CsEntry* entry = NULL;
  if (unlikely(pccEntry->hasCsEntry)) {
    // refresh direct entry
    entry = PccEntry_GetCsEntry(pccEntry);
    N_LOGD("PutDirect(refresh) cs=%p npkt=%p pcc-entry=%p cs-entry=%p", cs, npkt, pccEntry, entry);
    if (entry->kind != CsEntryIndirect) {
      Cs_EraseImplicitDigestIndirect(cs, entry, data->name.length);
    }
    CsEntry_Clear(entry);
  } else {
    // insert direct entry
    entry = PccEntry_AddCsEntry(pccEntry);
    if (unlikely(entry == NULL)) {
      N_LOGW("PutDirect alloc-err cs=%p npkt=%p pcc-entry=%p", cs, npkt, pccEntry);
      return NULL;
    }
    CsEntry_Init(entry);
    N_LOGD("PutDirect insert cs=%p npkt=%p pcc-entry=%p cs-entry=%p", cs, npkt, pccEntry, entry);
  }

  entry->kind = CsEntryMemory;
  entry->data = npkt;
  entry->freshUntil = Mbuf_GetTimestamp(pkt) + TscDuration_FromMillis(data->freshness);
  CsArc_Add(&cs->direct, entry);
  return entry;
}

/** @brief Insert a direct entry for @p npkt that was retrieved by @p interest . */
__attribute__((nonnull)) static CsEntry*
Cs_InsertDirect(Cs* cs, Packet* npkt, PInterest* interest)
{
  Pcct* pcct = Pcct_FromCs(cs);
  PData* data = Packet_GetDataHdr(npkt);

  // construct PccSearch
  PccSearch search = PccSearch_FromNames(&data->name, interest);

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
__attribute__((nonnull)) static bool
Cs_PutIndirect(Cs* cs, CsEntry* direct, PccEntry* pccEntry)
{
  NDNDPDK_ASSERT(!pccEntry->hasPitEntry0);

  CsEntry* entry = NULL;
  if (unlikely(pccEntry->hasCsEntry)) {
    entry = PccEntry_GetCsEntry(pccEntry);
    if (likely(entry->kind == CsEntryIndirect)) {
      // refresh indirect entry
      CsEntry_Disassoc(entry);
      CsList_MoveToLast(&cs->indirect, entry);
    } else if (unlikely(entry->nIndirects > 0)) {
      // don't overwrite direct entry with dependencies
      N_LOGD("PutIndirect has-dependency cs=%p npkt=%p pcc-entry-%p cs-entry=%p", cs, direct,
             pccEntry, entry);
      return false;
    } else {
      // change direct entry to indirect entry
      CsEntry_Clear(entry);
      CsList_Append(&cs->indirect, entry);
    }
    N_LOGD("PutIndirect refresh cs=%p npkt=%p pcc-entry-%p cs-entry=%p count=%" PRIu32, cs, direct,
           pccEntry, entry, cs->indirect.count);
  } else {
    // insert indirect entry
    entry = PccEntry_AddCsEntry(pccEntry);
    if (unlikely(entry == NULL)) {
      N_LOGW("PutIndirect alloc-err cs=%p npkt=%p pcc-entry-%p", cs, direct, pccEntry);
      return NULL;
    }
    CsEntry_Init(entry);
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
  if (likely(!pccEntry->hasEntries)) {
    Pcct_Erase(Pcct_FromCs(cs), pccEntry);
  }
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
    NULLize(npkt);
    NULLize(pkt); // owned by direct entry
  }

  // delete PIT entries
  Pit_RawErase01_(&pcct->pit, pccEntry);
  NULLize(interest);

  if (likely(direct == NULL)) {
    // put direct CS entry at pccEntry
    direct = Cs_PutDirect(cs, npkt, pccEntry);
    // alloc-err cannot happen because PccSlots are freed from PIT entries
    NDNDPDK_ASSERT(direct != NULL);
  } else {
    // put indirect CS entry at pccEntry
    Cs_PutIndirect(cs, direct, pccEntry);
  }

  // evict if over capacity
  if (unlikely(cs->indirect.count > cs->indirect.capacity)) {
    Cs_Evict(cs, &cs->indirect, "indirect", CsEraseBatch_AddIndirect);
  }
  if (unlikely(cs->direct.Del.count >= CsEvictBulk)) {
    Cs_Evict(cs, &cs->direct.Del, "direct", CsEraseBatch_AddDirect);
  }
}

bool
Cs_MatchInterest(Cs* cs, CsEntry* entry, Packet* interestNpkt)
{
  CsEntry* direct = CsEntry_GetDirect(entry);
  PccEntry* pccDirect = PccEntry_FromCsEntry(direct);

  PInterest* interest = Packet_GetInterestHdr(interestNpkt);
  bool violateCanBePrefix = !interest->canBePrefix && interest->name.length < pccDirect->key.nameL;
  bool violateMustBeFresh =
    interest->mustBeFresh && direct->freshUntil <= Mbuf_GetTimestamp(Packet_ToMbuf(interestNpkt));
  N_LOGD(
    "MatchInterest cs=%p cs-entry=%p entry-kind=%s direct-kind=%s violate-cbp=%d violate-mbf=%d",
    cs, entry, CsEntryKindString[entry->kind], CsEntryKindString[direct->kind],
    (int)violateCanBePrefix, (int)violateMustBeFresh);

  if (unlikely(violateCanBePrefix || violateMustBeFresh)) {
    return false;
  }

  switch (direct->kind) {
    case CsEntryNone:
      return false;
    case CsEntryMemory:
      CsArc_Add(&cs->direct, direct);
      ++cs->nHitMemory;
      break;
    case CsEntryDisk:
      ++cs->nHitDisk;
      return false; // XXX
    default:
      NDNDPDK_ASSERT(false);
      return false;
  }

  if (entry->kind == CsEntryIndirect) {
    CsList_MoveToLast(&cs->indirect, entry);
    ++cs->nHitIndirect;
  }
  return true;
}

void
Cs_Erase(Cs* cs, CsEntry* entry)
{
  N_LOGD("Erase cs=%p cs-entry=%p", cs, entry);
  Cs_EraseEntry(cs, entry);
}
