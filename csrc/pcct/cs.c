#include "cs.h"
#include "pit.h"

#include "../core/logger.h"

INIT_ZF_LOG(Cs);

// Bulk size of CS eviction, also the minimum CS capacity.
#define CS_EVICT_BULK 64

static void
CsEraseBatch_Append_(PcctEraseBatch* peb, CsEntry* entry, const char* isDirectDbg)
{
  PccEntry* pccEntry = PccEntry_FromCsEntry(entry);
  PccEntry_RemoveCsEntry(pccEntry);
  if (likely(!pccEntry->hasEntries)) {
    ZF_LOGD("^ cs=%p(%s) pcc=%p(erase)", entry, isDirectDbg, pccEntry);
    PcctEraseBatch_Append(peb, pccEntry);
  } else {
    ZF_LOGD("^ cs=%p(%s) pcc=%p(keep)", entry, isDirectDbg, pccEntry);
  }
}

/** \brief Erase an indirect CS entry.
 */
static void
CsEraseBatch_AddIndirect(PcctEraseBatch* peb, CsEntry* entry)
{
  assert(!CsEntry_IsDirect(entry));
  ZF_LOGV("^ indirect=%p direct=%p(%" PRId8 ")", entry, entry->direct, entry->direct->nIndirects);
  CsEntry_Finalize(entry);
  CsEraseBatch_Append_(peb, entry, "indirect");
}

/** \brief Erase a direct CS entry; delist and erase indirect entries.
 */
static void
CsEraseBatch_AddDirect(PcctEraseBatch* peb, CsEntry* entry)
{
  assert(CsEntry_IsDirect(entry));
  CsPriv* csp = Cs_GetPriv(Cs_FromPcct(peb->pcct));
  for (int i = 0; i < entry->nIndirects; ++i) {
    CsEntry* indirect = entry->indirect[i];
    CsList_Remove(&csp->indirectLru, indirect);
    CsEraseBatch_Append_(peb, indirect, "indirect-dep");
  }
  entry->nIndirects = 0;
  CsEntry_Finalize(entry);
  CsEraseBatch_Append_(peb, entry, "direct");
}

/** \brief Erase a CS entry including dependents.
 */
static void
Cs_Erase_(Cs* cs, CsEntry* entry)
{
  CsPriv* csp = Cs_GetPriv(cs);
  PcctEraseBatch peb = PcctEraseBatch_New(Cs_ToPcct(cs));
  if (CsEntry_IsDirect(entry)) {
    CsArc_Remove(&csp->directArc, entry);
    CsEraseBatch_AddDirect(&peb, entry);
  } else {
    CsList_Remove(&csp->indirectLru, entry);
    CsEraseBatch_AddIndirect(&peb, entry);
  }
  PcctEraseBatch_Finish(&peb);
}

static void
Cs_EvictBulk_(Cs* cs, CsList* csl, const char* cslName, CsList_EvictCb evictCb)
{
  ZF_LOGD("%p Evict(%s) count=%" PRIu32, cs, cslName, csl->count);
  PcctEraseBatch peb = PcctEraseBatch_New(Cs_ToPcct(cs));
  CsList_EvictBulk(csl, CS_EVICT_BULK, evictCb, &peb);
  PcctEraseBatch_Finish(&peb);
  ZF_LOGD("^ end-count=%" PRIu32, csl->count);
}

static void
Cs_Evict(Cs* cs)
{
  CsPriv* csp = Cs_GetPriv(cs);
  if (unlikely(csp->indirectLru.count > csp->indirectLru.capacity)) {
    Cs_EvictBulk_(cs, &csp->indirectLru, "indirect", (CsList_EvictCb)CsEraseBatch_AddIndirect);
  }
  if (unlikely(csp->directArc.DEL.count >= CS_EVICT_BULK)) {
    Cs_EvictBulk_(cs, &csp->directArc.DEL, "direct", (CsList_EvictCb)CsEraseBatch_AddDirect);
  }
}

static CsList*
CsPriv_GetList(CsPriv* csp, CsListId cslId)
{
  switch (cslId) {
    case CslMdT1:
    case CslMdB1:
    case CslMdT2:
    case CslMdB2:
    case CslMdDel:
      return CsArc_GetList(&csp->directArc, cslId - CslMd);
    case CslMi:
      return &csp->indirectLru;
    case CslMd:
    default:
      assert(false);
      return NULL;
  }
}

void
Cs_Init(Cs* cs, uint32_t capMd, uint32_t capMi)
{
  capMd = RTE_MAX(capMd, CS_EVICT_BULK);
  capMi = RTE_MAX(capMi, CS_EVICT_BULK);

  CsPriv* csp = Cs_GetPriv(cs);
  CsArc_Init(&csp->directArc, capMd);
  CsList_Init(&csp->indirectLru);
  csp->indirectLru.capacity = capMi;

  ZF_LOGI("%p Init() priv=%p cap-md=%" PRIu32 " cap-mi=%" PRIu32, cs, csp, capMd, capMi);
}

uint32_t
Cs_GetCapacity(const Cs* cs, CsListId cslId)
{
  CsPriv* csp = Cs_GetPriv(cs);
  if (cslId == CslMd) {
    return CsArc_GetCapacity(&csp->directArc);
  }
  return CsPriv_GetList(csp, cslId)->capacity;
}

uint32_t
Cs_CountEntries(const Cs* cs, CsListId cslId)
{
  CsPriv* csp = Cs_GetPriv(cs);
  if (cslId == CslMd) {
    return CsArc_CountEntries(&csp->directArc);
  }
  return CsPriv_GetList(csp, cslId)->count;
}

/** \brief Add or refresh a direct entry for \p npkt in \p pccEntry.
 */
static CsEntry*
Cs_PutDirect(Cs* cs, Packet* npkt, PccEntry* pccEntry)
{
  CsPriv* csp = Cs_GetPriv(cs);
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  PData* data = Packet_GetDataHdr(npkt);

  CsEntry* entry = NULL;
  if (unlikely(pccEntry->hasCsEntry)) {
    // refresh direct entry
    entry = PccEntry_GetCsEntry(pccEntry);
    ZF_LOGD("%p PutDirect(%p, pcc=%p) cs=%p refresh", cs, npkt, pccEntry, entry);
    if (CsEntry_IsDirect(entry)) {
      // erase any indirect entry with implicit digest name, because it may not match new Data
      for (int8_t i = 0; i < entry->nIndirects; ++i) {
        CsEntry* indirect = entry->indirect[i];
        PccEntry* indirectPcc = PccEntry_FromCsEntry(indirect);
        if (unlikely(indirectPcc->key.nameL > data->name.p.nOctets)) {
          ZF_LOGD("  ^ erase-implicit-digest-indirect");
          Cs_Erase_(cs, indirect);
          break;
        }
      }
    }
    CsEntry_Clear(entry);
    CsArc_Add(&csp->directArc, entry);
  } else {
    // insert direct entry
    entry = PccEntry_AddCsEntry(pccEntry);
    if (unlikely(entry == NULL)) {
      ZF_LOGW("%p PutDirect(%p, pcc=%p) drop=alloc-err", cs, npkt, pccEntry);
      return NULL;
    }
    ZF_LOGD("%p PutDirect(%p, pcc=%p) cs=%p insert", cs, npkt, pccEntry, entry);
    entry->arcList = CSL_ARC_NONE;
    entry->nIndirects = 0;
    CsArc_Add(&csp->directArc, entry);
  }
  entry->data = npkt;
  entry->freshUntil = pkt->timestamp + TscDuration_FromMillis(data->freshnessPeriod);
  return entry;
}

/** \brief Insert a direct entry for \p npkt that was retrieved by \p interest.
 */
static CsEntry*
Cs_InsertDirect(Cs* cs, Packet* npkt, PInterest* interest)
{
  Pcct* pcct = Cs_ToPcct(cs);
  PData* data = Packet_GetDataHdr(npkt);

  // construct PccSearch
  PccSearch search;
  PccSearch_FromNames(&search, &data->name, interest);

  // seek PCC entry
  bool isNewPcc = false;
  PccEntry* pccEntry = Pcct_Insert(pcct, &search, &isNewPcc);
  if (unlikely(pccEntry == NULL)) {
    ZF_LOGD("%p InsertDirect(%p) drop=alloc-err", cs, npkt);
    return NULL;
  }

  // put direct entry on PCC entry
  return Cs_PutDirect(cs, npkt, pccEntry);
}

/** \brief Add or refresh an indirect entry in \p pccEntry and associate with \p direct.
 */
static bool
Cs_PutIndirect(Cs* cs, CsEntry* direct, PccEntry* pccEntry)
{
  assert(!pccEntry->hasPitEntry0);
  CsPriv* csp = Cs_GetPriv(cs);

  CsEntry* entry = NULL;
  if (unlikely(pccEntry->hasCsEntry)) {
    entry = PccEntry_GetCsEntry(pccEntry);
    if (unlikely(CsEntry_IsDirect(entry) && entry->nIndirects > 0)) {
      // don't overwrite direct entry with dependencies
      ZF_LOGD("%p PutIndirect(%p, pcc=%p) cs=%p drop=has-dependency", cs, direct, pccEntry, entry);
      return false;
    }
    // refresh indirect entry
    // old entry can be either direct without dependency or indirect
    CsEntry_Clear(entry);
    CsList_MoveToLast(&csp->indirectLru, entry);
    ZF_LOGD("%p PutIndirect(%p, pcc=%p) cs=%p count=%" PRIu32 " refresh", cs, direct, pccEntry,
            entry, csp->indirectLru.count);
  } else {
    // insert indirect entry
    entry = PccEntry_AddCsEntry(pccEntry);
    if (unlikely(entry == NULL)) {
      ZF_LOGW("%p PutIndirect(%p, pcc=%p) drop=alloc-err", cs, direct, pccEntry);
      return NULL;
    }
    entry->nIndirects = 0;
    CsList_Append(&csp->indirectLru, entry);
    ZF_LOGD("%p PutIndirect(%p, pcc=%p) cs=%p count=%" PRIu32 " insert", cs, direct, pccEntry,
            entry, csp->indirectLru.count);
  }

  if (likely(CsEntry_Assoc(entry, direct))) {
    ZF_LOGV("^ indirect=%p direct=%p(%" PRId8 ")", entry, entry->direct, entry->direct->nIndirects);
    return true;
  }

  ZF_LOGD("^ drop=indirect-assoc-err");
  CsList_Remove(&csp->indirectLru, entry);
  PccEntry_RemoveCsEntry(pccEntry);
  Pcct_Erase(Cs_ToPcct(cs), pccEntry);
  return false;
}

void
Cs_Insert(Cs* cs, Packet* npkt, PitFindResult pitFound)
{
  Pcct* pcct = Cs_ToPcct(cs);
  Pit* pit = Pit_FromPcct(pcct);
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  PData* data = Packet_GetDataHdr(npkt);
  PccEntry* pccEntry = pitFound.entry;
  PInterest* interest = PitFindResult_GetInterest_(pitFound);
  CsEntry* direct = NULL;

  // if Interest name differs from Data name, insert a direct entry elsewhere
  if (unlikely(interest->name.p.nComps != data->name.p.nComps)) {
    direct = Cs_InsertDirect(cs, npkt, interest);
    if (unlikely(direct == NULL)) { // direct entry insertion failed
      Pit_RawErase01_(pit, pccEntry);
      rte_pktmbuf_free(pkt);
      if (likely(!pccEntry->hasCsEntry)) {
        Pcct_Erase(pcct, pccEntry);
      }
      return;
    }
    pkt = NULL; // owned by direct entry, don't free it
  }

  // delete PIT entries
  Pit_RawErase01_(pit, pccEntry);
  interest = NULL;

  if (likely(direct == NULL)) {
    // put direct CS entry at pccEntry
    direct = Cs_PutDirect(cs, npkt, pccEntry);
    assert(direct != NULL);
  } else {
    // put indirect CS entry at pccEntry
    Cs_PutIndirect(cs, direct, pccEntry);
  }

  // evict if over capacity
  Cs_Evict(cs);
}

bool
Cs_MatchInterest_(Cs* cs, PccEntry* pccEntry, Packet* interestNpkt)
{
  CsEntry* entry = PccEntry_GetCsEntry(pccEntry);
  CsEntry* direct = CsEntry_GetDirect(entry);
  bool hasData = CsEntry_GetData(direct) != NULL;
  PccEntry* pccDirect = PccEntry_FromCsEntry(direct);

  PInterest* interest = Packet_GetInterestHdr(interestNpkt);
  bool violateCanBePrefix =
    !interest->canBePrefix && interest->name.p.nOctets < pccDirect->key.nameL;
  bool violateMustBeFresh =
    interest->mustBeFresh && !CsEntry_IsFresh(direct, Packet_ToMbuf(interestNpkt)->timestamp);
  ZF_LOGD("%p MatchInterest(%p,cs=%p~%s) cbp=%s mbf=%s has-data=%s", cs, pccEntry, entry,
          CsEntry_IsDirect(entry) ? "direct" : "indirect", violateCanBePrefix ? "N" : "Y",
          violateMustBeFresh ? "N" : "Y", hasData ? "Y" : "N");

  if (likely(!violateCanBePrefix && !violateMustBeFresh)) {
    CsPriv* csp = Cs_GetPriv(cs);
    if (!CsEntry_IsDirect(entry)) {
      CsList_MoveToLast(&csp->indirectLru, entry);
    }
    if (likely(hasData)) {
      CsArc_Add(&csp->directArc, direct);
      return true;
    }
  }
  return false;
}

void
Cs_Erase(Cs* cs, CsEntry* entry)
{
  ZF_LOGD("%p Erase(%p)", cs, entry);
  Cs_Erase_(cs, entry);
}
