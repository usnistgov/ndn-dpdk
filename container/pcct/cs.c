#include "cs.h"
#include "pit.h"

#include "../../core/logger.h"

INIT_ZF_LOG(Cs);

// Bulk size of CS eviction, also the minimum CS capacity.
#define CS_EVICT_BULK 64

/** \brief Context for erasing several CS entries.
 */
typedef struct CsEraseBatch
{
  Cs* cs;
  uint32_t nPccErase;
  PccEntry* pccErase[CS_EVICT_BULK * (1 + CS_ENTRY_MAX_INDIRECTS)];
} CsEraseBatch;

#define CsEraseBatch_New(theCs)                                                \
  {                                                                            \
    0, .cs = theCs                                                             \
  }

static void
__CsEraseBatch_Append(CsEraseBatch* ceb, CsEntry* entry, bool wantKeepPcc,
                      const char* isDirectDbg)
{
  PccEntry* pccEntry = PccEntry_FromCsEntry(entry);
  if (likely(!pccEntry->hasPitEntry1 && !wantKeepPcc)) {
    assert(ceb->nPccErase < RTE_DIM(ceb->pccErase));
    ceb->pccErase[ceb->nPccErase++] = pccEntry;
    ZF_LOGD("^ cs=%p(%s) pcc=%p(erase)", entry, isDirectDbg, pccEntry);
  } else {
    pccEntry->hasCsEntry = false;
    ZF_LOGD("^ cs=%p(%s) pcc=%p(keep)", entry, isDirectDbg, pccEntry);
  }
}

/** \brief Erase an indirect CS entry.
 *  \param wantKeepPcc if true, PCC entry for \p entry is not erased.
 */
static void
CsEraseBatch_AddIndirect(CsEraseBatch* ceb, CsEntry* entry, bool wantKeepPcc)
{
  assert(!CsEntry_IsDirect(entry));
  ZF_LOGV("^ indirect=%p direct=%p(%" PRId8 ")", entry, entry->direct,
          entry->direct->nIndirects);
  CsEntry_Finalize(entry);
  __CsEraseBatch_Append(ceb, entry, wantKeepPcc, "indirect");
}

/** \brief Erase an indirect CS entry.
 *  \param ceb0 pointer to CsEraseBatch.
 */
static void
CsEraseBatch_EvictIndirect(void* ceb0, CsEntry* entry)
{
  CsEraseBatch* ceb = (CsEraseBatch*)ceb0;
  CsEraseBatch_AddIndirect(ceb, entry, false);
}

/** \brief Erase a direct CS entry; delist and erase indirect entries.
 *  \param wantKeepSelfPcc if true, PCC entry for \p entry is not erased.
 */
static void
CsEraseBatch_AddDirect(CsEraseBatch* ceb, CsEntry* entry, bool wantKeepSelfPcc)
{
  assert(CsEntry_IsDirect(entry));
  CsPriv* csp = Cs_GetPriv(ceb->cs);
  for (int i = 0; i < entry->nIndirects; ++i) {
    CsEntry* indirect = entry->indirect[i];
    CsList_Remove(&csp->indirectLru, indirect);
    __CsEraseBatch_Append(ceb, indirect, false, "indirect-dep");
  }
  entry->nIndirects = 0;
  CsEntry_Finalize(entry);
  __CsEraseBatch_Append(ceb, entry, wantKeepSelfPcc, "direct");
}

/** \brief Erase a direct CS entry; delist and erase indirect entries.
 *  \param ceb0 pointer to CsEraseBatch.
 */
static void
CsEraseBatch_EvictDirect(void* ceb0, CsEntry* entry)
{
  CsEraseBatch* ceb = (CsEraseBatch*)ceb0;
  CsEraseBatch_AddDirect(ceb, entry, false);
}

/** \brief Remove an entry from CsList and erase it including dependents.
 *  \param wantKeepSelfPcc if true, PCC entry for \p entry is not erased.
 */
static void
CsEraseBatch_DelistAndErase(CsEraseBatch* ceb, CsEntry* entry,
                            bool wantKeepSelfPcc)
{
  CsPriv* csp = Cs_GetPriv(ceb->cs);
  if (CsEntry_IsDirect(entry)) {
    CsArc_Remove(&csp->directArc, entry);
    CsEraseBatch_AddDirect(ceb, entry, wantKeepSelfPcc);
  } else {
    CsList_Remove(&csp->indirectLru, entry);
    CsEraseBatch_AddIndirect(ceb, entry, wantKeepSelfPcc);
  }
}

/** \brief Erase empty PCC entries used by erased CS entries.
 */
static bool
CsEraseBatch_Finish(CsEraseBatch* ceb)
{
  Pcct_EraseBulk(Cs_ToPcct(ceb->cs), ceb->pccErase, ceb->nPccErase);
}

static void
__Cs_EvictBulk(Cs* cs, CsList* csl, const char* cslName, CsList_EvictCb evictCb)
{
  ZF_LOGD("%p Evict(%s) count=%" PRIu32, cs, cslName, csl->count);
  CsEraseBatch ceb = CsEraseBatch_New(cs);
  CsList_EvictBulk(csl, CS_EVICT_BULK, evictCb, &ceb);
  CsEraseBatch_Finish(&ceb);
  ZF_LOGD("^ end-count=%" PRIu32, csl->count);
}

static void
Cs_Evict(Cs* cs)
{
  CsPriv* csp = Cs_GetPriv(cs);
  if (unlikely(csp->indirectLru.count > csp->indirectLru.capacity)) {
    __Cs_EvictBulk(cs, &csp->indirectLru, "indirect",
                   CsEraseBatch_EvictIndirect);
  }
  if (unlikely(csp->directArc.DEL.count >= CS_EVICT_BULK)) {
    __Cs_EvictBulk(cs, &csp->directArc.DEL, "direct", CsEraseBatch_EvictDirect);
  }
}

static CsList*
CsPriv_GetList(CsPriv* csp, CsListId cslId)
{
  switch (cslId) {
    case CSL_MD_T1:
    case CSL_MD_B1:
    case CSL_MD_T2:
    case CSL_MD_B2:
    case CSL_MD_DEL:
      return CsArc_GetList(&csp->directArc, cslId - CSL_MD);
    case CSL_MI:
      return &csp->indirectLru;
    case CSL_MD:
    default:
      assert(false);
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

  ZF_LOGI("%p Init() priv=%p cap-md=%" PRIu32 " cap-mi=%" PRIu32, cs, csp,
          capMd, capMi);
}

uint32_t
Cs_GetCapacity(const Cs* cs, CsListId cslId)
{
  CsPriv* csp = Cs_GetPriv(cs);
  if (cslId == CSL_MD) {
    return CsArc_GetCapacity(&csp->directArc);
  }
  return CsPriv_GetList(csp, cslId)->capacity;
}

uint32_t
Cs_CountEntries(const Cs* cs, CsListId cslId)
{
  CsPriv* csp = Cs_GetPriv(cs);
  if (cslId == CSL_MD) {
    return CsArc_CountEntries(&csp->directArc);
  }
  return CsPriv_GetList(csp, cslId)->count;
}

static bool
Cs_PutDirect(Cs* cs, Packet* npkt, PccEntry* pccEntry)
{
  CsPriv* csp = Cs_GetPriv(cs);
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  PData* data = Packet_GetDataHdr(npkt);

  CsEntry* entry = &pccEntry->csEntry;
  if (unlikely(pccEntry->hasCsEntry)) {
    // refresh direct entry
    // old entry can be either direct or indirect
    // XXX If old entry is direct, and an indirect entry with full name (incl
    // implicit digest) depends on it, refreshing with a different Data could
    // change the implicit digest, and cause that indirect entry to become
    // non-matching. This code does not handle this case correctly.
    CsEntry_Clear(entry);
    CsArc_Add(&csp->directArc, entry);
    ZF_LOGD("%p PutDirect(%p, pcc=%p) cs=%p refresh", cs, npkt, pccEntry,
            entry);
  } else if (unlikely(pccEntry->hasPitEntry0)) {
    ZF_LOGD("%p PutDirect(%p, pcc=%p) drop=has-pit0", cs, npkt, pccEntry);
    return false;
  } else {
    // insert direct entry
    pccEntry->hasCsEntry = true;
    entry->arcList = CSL_ARC_NONE;
    entry->nIndirects = 0;
    CsArc_Add(&csp->directArc, entry);
    ZF_LOGD("%p PutDirect(%p, pcc=%p) cs=%p insert", cs, npkt, pccEntry, entry);
  }
  entry->data = npkt;
  entry->freshUntil =
    pkt->timestamp + TscDuration_FromMillis(data->freshnessPeriod);
  return true;
}

static CsEntry*
Cs_InsertDirect(Cs* cs, Packet* npkt, PInterest* interest)
{
  Pcct* pcct = Cs_ToPcct(cs);
  PData* data = Packet_GetDataHdr(npkt);

  // construct PccSearch
  PccSearch search = { 0 };
  search.name = *(const LName*)(&data->name);
  search.nameHash = PName_ComputeHash(&data->name.p, data->name.v);
  if (interest->activeFh >= 0) {
    search.fh = *(const LName*)(&interest->activeFhName);
    search.fhHash =
      PName_ComputeHash(&interest->activeFhName.p, interest->activeFhName.v);
  }

  // seek PCC entry
  bool isNewPcc = false;
  PccEntry* pccEntry = Pcct_Insert(pcct, &search, &isNewPcc);
  if (unlikely(pccEntry == NULL)) {
    ZF_LOGD("%p InsertDirect(%p) drop=alloc-err", cs, npkt);
    return NULL;
  }

  // put direct entry on PCC entry
  if (likely(Cs_PutDirect(cs, npkt, pccEntry))) {
    return PccEntry_GetCsEntry(pccEntry);
  }
  return NULL;
}

static bool
Cs_PutIndirect(Cs* cs, CsEntry* direct, PccEntry* pccEntry)
{
  assert(!pccEntry->hasPitEntry0);
  CsPriv* csp = Cs_GetPriv(cs);

  CsEntry* entry = &pccEntry->csEntry;
  if (unlikely(pccEntry->hasCsEntry)) {
    if (unlikely(CsEntry_IsDirect(entry) && entry->nIndirects > 0)) {
      // don't overwrite direct entry with dependencies
      ZF_LOGD("%p PutIndirect(%p, pcc=%p) cs=%p drop=has-dependency", cs,
              direct, pccEntry, entry);
      return false;
    }
    // refresh indirect entry
    // old entry can be either direct without dependency or indirect
    CsEntry_Clear(entry);
    CsList_MoveToLast(&csp->indirectLru, entry);
    ZF_LOGD("%p PutIndirect(%p, pcc=%p) cs=%p count=%" PRIu32 " refresh", cs,
            direct, pccEntry, entry, csp->indirectLru.count);
  } else {
    // insert indirect entry
    pccEntry->hasCsEntry = true;
    entry->nIndirects = 0;
    CsList_Append(&csp->indirectLru, entry);
    ZF_LOGD("%p PutIndirect(%p, pcc=%p) cs=%p count=%" PRIu32 " insert", cs,
            direct, pccEntry, entry, csp->indirectLru.count);
  }

  if (likely(CsEntry_Assoc(entry, direct))) {
    ZF_LOGV("^ indirect=%p direct=%p(%" PRId8 ")", entry, entry->direct,
            entry->direct->nIndirects);
    return true;
  }

  ZF_LOGD("^ drop=indirect-assoc-err");
  CsList_Remove(&csp->indirectLru, entry);
  pccEntry->hasCsEntry = false;
  Pcct_Erase(Cs_ToPcct(cs), pccEntry);
  return false;
}

void
Cs_Insert(Cs* cs, Packet* npkt, PitFindResult pitFound)
{
  Pcct* pcct = Cs_ToPcct(cs);
  Pit* pit = Pit_FromPcct(pcct);
  CsPriv* csp = Cs_GetPriv(cs);
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  PData* data = Packet_GetDataHdr(npkt);
  PccEntry* pccEntry = pitFound.entry;
  PInterest* interest = __PitFindResult_GetInterest(pitFound);
  CsEntry* direct = NULL;

  // if Interest name is shorter or longer than Data name, insert a direct CS
  // entry in another PCC entry, and put an indirect CS entry at pccEntry
  if (unlikely(interest->name.p.nComps != data->name.p.nComps)) {
    direct = Cs_InsertDirect(cs, npkt, interest);
    if (unlikely(direct == NULL)) {
      __Pit_RawErase01(pit, pccEntry);
      rte_pktmbuf_free(pkt);
      if (likely(!pccEntry->hasCsEntry)) {
        Pcct_Erase(pcct, pccEntry);
      }
      return;
    }
    pkt = NULL; // owned by direct entry, don't free it
  }

  // delete PIT entries
  __Pit_RawErase01(pit, pccEntry);
  interest = NULL;

  if (likely(direct == NULL)) {
    // put direct CS entry at pccEntry
    bool ok = Cs_PutDirect(cs, npkt, pccEntry);
    assert(ok);
  } else {
    // put indirect CS entry at pccEntry
    Cs_PutIndirect(cs, direct, pccEntry);
  }

  // evict if over capacity
  Cs_Evict(cs);
}

bool
__Cs_MatchInterest(Cs* cs, PccEntry* pccEntry, Packet* interestNpkt)
{
  CsEntry* entry = PccEntry_GetCsEntry(pccEntry);
  CsEntry* direct = CsEntry_GetDirect(entry);
  bool hasData = CsEntry_GetData(direct) != NULL;
  PccEntry* pccDirect = PccEntry_FromCsEntry(direct);

  PInterest* interest = Packet_GetInterestHdr(interestNpkt);
  bool violateCanBePrefix =
    !interest->canBePrefix && interest->name.p.nOctets < pccDirect->key.nameL;
  bool violateMustBeFresh =
    interest->mustBeFresh &&
    !CsEntry_IsFresh(direct, Packet_ToMbuf(interestNpkt)->timestamp);
  ZF_LOGD("%p MatchInterest(%p,cs=%p~%s) cbp=%s mbf=%s has-data=%s", cs,
          pccEntry, entry, CsEntry_IsDirect(entry) ? "direct" : "indirect",
          violateCanBePrefix ? "N" : "Y", violateMustBeFresh ? "N" : "Y",
          hasData ? "Y" : "N");

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

  if (!interest->mustBeFresh) {
    // erase CS entry to make room for pitEntry0
    ZF_LOGD("%p MatchInterest(%p) erase-conflict-PIT cs=%p", cs, pccEntry,
            entry);
    CsEraseBatch ceb = CsEraseBatch_New(cs);
    CsEraseBatch_DelistAndErase(&ceb, entry, true);
    CsEraseBatch_Finish(&ceb);
  }
  return false;
}

void
Cs_Erase(Cs* cs, CsEntry* entry)
{
  ZF_LOGD("%p Erase(%p)", cs, entry);
  CsEraseBatch ceb = CsEraseBatch_New(cs);
  CsEraseBatch_DelistAndErase(&ceb, entry, false);
  CsEraseBatch_Finish(&ceb);
}
