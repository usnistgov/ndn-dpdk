#include "cs.h"
#include "cs-disk.h"
#include "pit.h"

#include "../core/logger.h"

N_LOG_INIT(Cs);

__attribute__((nonnull)) static void
CsEraseBatch_Append(PcctEraseBatch* peb, CsEntry* entry, const char* kind) {
  PccEntry* pccEntry = entry->pccEntry;
  PccEntry_RemoveCsEntry(pccEntry);
  if (likely(!pccEntry->hasEntries)) {
    N_LOGD("^ cs=%p(%s) pcc=%p(erase)", entry, kind, pccEntry);
    PcctEraseBatch_Append(peb, pccEntry);
  } else {
    N_LOGD("^ cs=%p(%s) pcc=%p(keep)", entry, kind, pccEntry);
  }
}

/** @brief Erase an indirect CS entry. */
__attribute__((nonnull)) static void
CsEraseBatch_AddIndirect(PcctEraseBatch* peb, CsEntry* entry) {
  N_LOGV("^ indirect=%p direct=%p(%" PRId8 ")", entry, entry->direct, entry->direct->nIndirects);
  CsEntry_Disassoc(entry);
  CsEraseBatch_Append(peb, entry, "indirect");
}

/** @brief Erase a direct CS entry; delist and erase indirect entries. */
__attribute__((nonnull)) static void
CsEraseBatch_AddDirect(PcctEraseBatch* peb, CsEntry* entry) {
  Cs* cs = &peb->pcct->cs;
  for (int i = 0; i < entry->nIndirects; ++i) {
    CsEntry* indirect = entry->indirect[i];
    // skip CsEntry_Disassoc because the direct entry is being released
    CsList_Remove(&cs->indirect, indirect);
    CsEraseBatch_Append(peb, indirect, "indirect-dep");
  }

  NDNDPDK_ASSERT(entry->kind == CsEntryNone);
  CsEraseBatch_Append(peb, entry, "direct");
}

/** @brief Erase a CS entry including dependents. */
__attribute__((nonnull)) static void
Cs_EraseEntry(Cs* cs, CsEntry* entry) {
  PcctEraseBatch peb = PcctEraseBatch_New(Pcct_FromCs(cs));
  if (entry->kind == CsEntryIndirect) {
    CsList_Remove(&cs->indirect, entry);
    CsEraseBatch_AddIndirect(&peb, entry);
  } else {
    CsArc_Remove(&cs->direct, entry);
    CsEraseBatch_AddDirect(&peb, entry);
  }
  PcctEraseBatch_Finish(&peb);
}

__attribute__((nonnull)) static void
Cs_EvictEntryIndirect(CsEntry* entry, uintptr_t ctx) {
  CsEraseBatch_AddIndirect((PcctEraseBatch*)ctx, entry);
}

__attribute__((nonnull)) static void
Cs_EvictEntryDirect(CsEntry* entry, uintptr_t ctx) {
  CsEraseBatch_AddDirect((PcctEraseBatch*)ctx, entry);
}

__attribute__((nonnull)) static void
Cs_Evict(Cs* cs, CsList* csl, const char* cslName, CsList_EvictCb evictCb) {
  N_LOGD("%p Evict(%s) count=%" PRIu32, cs, cslName, csl->count);
  PcctEraseBatch peb = PcctEraseBatch_New(Pcct_FromCs(cs));
  CsList_EvictBulk(csl, CsEvictBulk, evictCb, (uintptr_t)&peb);
  PcctEraseBatch_Finish(&peb);
  N_LOGD("^ end-count=%" PRIu32, csl->count);
}

__attribute__((nonnull, returns_nonnull)) static inline CsList*
Cs_GetList(Cs* cs, CsListID l) {
  if (l == CslIndirect) {
    return &cs->indirect;
  }
  return CsArc_GetList(&cs->direct, l);
}

uint32_t
Cs_GetCapacity(Cs* cs, CsListID l) {
  if (l == CslDirect) {
    return CsArc_GetCapacity(&cs->direct);
  }
  return Cs_GetList(cs, l)->capacity;
}

uint32_t
Cs_CountEntries(Cs* cs, CsListID l) {
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
Cs_EraseImplicitDigestIndirect(Cs* cs, CsEntry* direct, size_t dataNameL) {
  for (uint8_t i = 0; i < direct->nIndirects; ++i) {
    CsEntry* indirect = direct->indirect[i];
    if (unlikely(indirect->pccEntry->key.nameL > dataNameL)) {
      N_LOGD("^ erase-implicit-digest-indirect indirect=%p direct=%p", indirect, direct);
      Cs_EraseEntry(cs, indirect);
      break;
    }
  }
}

/** @brief Add or refresh a direct entry for @p npkt in @p pccEntry . */
__attribute__((nonnull)) static CsEntry*
Cs_PutDirect(Cs* cs, Packet* npkt, PccEntry* pccEntry) {
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  PData* data = Packet_GetDataHdr(npkt);

  CsEntry* entry = NULL;
  if (unlikely(pccEntry->hasCsEntry)) {
    // refresh direct entry
    entry = PccEntry_GetCsEntry(pccEntry);
    N_LOGD("PutDirect(refresh) cs=%p npkt=%p pcc-entry=%p cs-entry=%p", cs, npkt, pccEntry, entry);
    switch (entry->kind) {
      case CsEntryMemory:
        CsEntry_FreeData(entry);
        // fallthrough
      case CsEntryDisk:
        // diskSlot will be released by CsArc_Add that invokes CsDisk_ArcMove
        // fallthrough
      case CsEntryNone:
        Cs_EraseImplicitDigestIndirect(cs, entry, data->name.length);
        break;
      case CsEntryIndirect:
        CsEntry_Disassoc(entry);
        break;
    }
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

  CsArc_Add(&cs->direct, entry);
  entry->kind = CsEntryMemory;
  entry->data = npkt;
  entry->freshUntil = Mbuf_GetTimestamp(pkt) + TscDuration_FromMillis(data->freshness);
  return entry;
}

/** @brief Insert a direct entry for @p npkt that was retrieved by @p interest . */
__attribute__((nonnull)) static CsEntry*
Cs_InsertDirect(Cs* cs, Packet* npkt, PInterest* interest) {
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
Cs_PutIndirect(Cs* cs, CsEntry* direct, PccEntry* pccEntry) {
  CsEntry* entry = NULL;
  if (unlikely(pccEntry->hasCsEntry)) {
    entry = PccEntry_GetCsEntry(pccEntry);
    if (likely(entry->kind == CsEntryIndirect)) {
      // refresh indirect entry
      CsEntry_Disassoc(entry);
      CsList_Remove(&cs->indirect, entry);
    } else if (unlikely(entry->nIndirects > 0)) {
      // don't overwrite direct entry that has dependencies
      N_LOGD("PutIndirect has-dependency cs=%p npkt=%p pcc-entry=%p cs-entry=%p", cs, direct,
             pccEntry, entry);
      return false;
    } else {
      // change direct entry to indirect entry
      CsArc_Remove(&cs->direct, entry);
    }
    N_LOGD("PutIndirect refresh cs=%p npkt=%p pcc-entry=%p cs-entry=%p", cs, direct, pccEntry,
           entry);
  } else {
    // insert indirect entry
    entry = PccEntry_AddCsEntry(pccEntry);
    if (unlikely(entry == NULL)) {
      N_LOGW("PutIndirect alloc-err cs=%p npkt=%p pcc-entry=%p", cs, direct, pccEntry);
      return false;
    }
    CsEntry_Init(entry);
    N_LOGD("PutIndirect insert cs=%p npkt=%p pcc-entry-%p cs-entry=%p", cs, direct, pccEntry,
           entry);
  }

  if (likely(CsEntry_Assoc(entry, direct))) {
    CsList_Append(&cs->indirect, entry);
    N_LOGV("^ count=%" PRIu32 " indirect=%p direct=%p(%" PRId8 ")", cs->indirect.count, entry,
           direct, direct->nIndirects);
    return true;
  }

  N_LOGD("^ indirect-assoc-err direct=%p(%" PRId8 ")", direct, direct->nIndirects);
  PccEntry_RemoveCsEntry(pccEntry);
  if (likely(!pccEntry->hasEntries)) {
    Pcct_Erase(Pcct_FromCs(cs), pccEntry);
  }
  return false;
}

void
Cs_Insert(Cs* cs, Packet* npkt, PitFindResult pitFound) {
  Pcct* pcct = Pcct_FromCs(cs);
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  PData* data = Packet_GetDataHdr(npkt);
  PccEntry* pccEntry = pitFound.entry;
  PInterest* interest = PitFindResult_GetInterest(pitFound);

  if (interest->name.nComps == data->name.nComps) { // exact match, direct entry here
    Pit_EraseSatisfied(&pcct->pit, pitFound);
    NULLize(interest);

    CsEntry* direct = Cs_PutDirect(cs, npkt, pccEntry);
    if (unlikely(direct == NULL)) {
      goto FAIL_DIRECT;
    }
  } else { // prefix match, indirect entry here, direct entry elsewhere
    CsEntry* direct = Cs_InsertDirect(cs, npkt, interest);
    Pit_EraseSatisfied(&pcct->pit, pitFound);
    NULLize(interest);

    if (unlikely(direct == NULL)) {
      goto FAIL_DIRECT;
    }
    Cs_PutIndirect(cs, direct, pccEntry);
  }
  NULLize(npkt); // owned by direct entry
  NULLize(pkt);

  // evict if over capacity
  if (unlikely(cs->indirect.count > cs->indirect.capacity)) {
    Cs_Evict(cs, &cs->indirect, "indirect", Cs_EvictEntryIndirect);
  }
  if (unlikely(cs->direct.Del.count >= CsEvictBulk)) {
    Cs_Evict(cs, &cs->direct.Del, "direct", Cs_EvictEntryDirect);
  }

  return;
FAIL_DIRECT:
  rte_pktmbuf_free(pkt);
  if (likely(!pccEntry->hasEntries)) {
    Pcct_Erase(pcct, pccEntry);
  }
}

CsEntry*
Cs_MatchInterest(Cs* cs, CsEntry* entry, Packet* interestNpkt) {
  CsEntry* direct = CsEntry_GetDirect(entry);
  PInterest* interest = Packet_GetInterestHdr(interestNpkt);
  bool violateCanBePrefix =
    !interest->canBePrefix && interest->name.length < direct->pccEntry->key.nameL;
  bool violateMustBeFresh =
    interest->mustBeFresh && direct->freshUntil <= Mbuf_GetTimestamp(Packet_ToMbuf(interestNpkt));
  N_LOGD(
    "MatchInterest cs=%p cs-entry=%p entry-kind=%s direct-kind=%s violate-cbp=%d violate-mbf=%d",
    cs, entry, CsEntryKind_ToString(entry->kind), CsEntryKind_ToString(direct->kind),
    (int)violateCanBePrefix, (int)violateMustBeFresh);

  if (unlikely(violateCanBePrefix || violateMustBeFresh)) {
    return NULL;
  }

  switch (direct->kind) {
    case CsEntryNone:
      return NULL;
    case CsEntryMemory:
      CsArc_Add(&cs->direct, direct);
      ++cs->nHitMemory;
      break;
    case CsEntryDisk:
      if (interest->diskSlot == direct->diskSlot) {
        interest->diskSlot = 0;
        if (unlikely(interest->diskData == NULL)) {
          N_LOGD("^ disk-slot=%" PRIu64 " disk-data-npkt=%p change-kind-as=none", direct->diskSlot,
                 interest->diskData);
          CsDisk_Delete(cs, direct);
          return NULL;
        } else {
          N_LOGD("^ disk-slot=%" PRIu64 " disk-data-npkt=%p change-kind-as=memory",
                 direct->diskSlot, interest->diskData);
          CsArc_Add(&cs->direct, direct);
          direct->kind = CsEntryMemory;
          direct->data = interest->diskData;
          interest->diskData = NULL;
          // not counting cache hit because it's a continuation of previous cache hit
        }
      } else {
        if (unlikely(interest->diskSlot != 0)) {
          interest->diskSlot = 0;
          rte_pktmbuf_free(Packet_ToMbuf(interest->diskData));
          interest->diskData = NULL;
        }
        ++cs->nHitDisk;
      }
      break;
    default:
      NDNDPDK_ASSERT(false);
      return NULL;
  }

  if (entry->kind == CsEntryIndirect) {
    CsList_MoveToLast(&cs->indirect, entry);
    ++cs->nHitIndirect;
  }
  return direct;
}

void
Cs_Erase(Cs* cs, CsEntry* entry) {
  N_LOGD("Erase cs=%p cs-entry=%p", cs, entry);
  Cs_EraseEntry(cs, entry);
}
