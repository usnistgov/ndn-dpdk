#include "cs.h"
#include "pit.h"

#include "../../core/logger.h"

INIT_ZF_LOG(Cs);

// Bulk size of CS eviction, also the minimum CS capacity.
#define CS_EVICT_BULK 64
static void Cs_Evict(Cs* cs);

static CsList*
CsPriv_GetList(CsPriv* csp, CsListId cslId)
{
  switch (cslId) {
    case CSL_MD:
      return &csp->directFifo;
    case CSL_MI:
      return &csp->indirectFifo;
  }
  assert(false);
}

void
Cs_Init(Cs* cs)
{
  CsPriv* csp = Cs_GetPriv(cs);
  CsList_Init(&csp->directFifo);
  CsList_Init(&csp->indirectFifo);

  csp->directFifo.capacity = CS_EVICT_BULK;
  csp->indirectFifo.capacity = CS_EVICT_BULK;

  ZF_LOGI("%p Init() priv=%p", cs, csp);
}

uint32_t
Cs_GetCapacity(const Cs* cs, CsListId cslId)
{
  CsPriv* csp = Cs_GetPriv(cs);
  return CsPriv_GetList(csp, cslId)->capacity;
}

void
Cs_SetCapacity(Cs* cs, CsListId cslId, uint32_t capacity)
{
  CsPriv* csp = Cs_GetPriv(cs);
  capacity = RTE_MAX(capacity, CS_EVICT_BULK);
  CsPriv_GetList(csp, cslId)->capacity = capacity;
  ZF_LOGI("%p SetCapacity(%s, %" PRIu32 ")", cs, CsListId_GetName(cslId),
          capacity);

  Cs_Evict(cs);
}

uint32_t
Cs_CountEntries(const Cs* cs, CsListId cslId)
{
  CsPriv* csp = Cs_GetPriv(cs);
  return CsPriv_GetList(csp, cslId)->count;
}

static bool
__Cs_PutDirect(Cs* cs, Packet* npkt, PccEntry* pccEntry)
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
    CsList_MoveToLast(&csp->directFifo, entry);
    ZF_LOGD("%p PutDirect(%p, pcc=%p) cs=%p count=%" PRIu32 " refresh", cs,
            npkt, pccEntry, entry, csp->directFifo.count);
  } else if (unlikely(pccEntry->hasPitEntry0)) {
    ZF_LOGD("%p PutDirect(%p, pcc=%p) drop=has-pit0", cs, npkt, pccEntry);
    return false;
  } else {
    // insert direct entry
    pccEntry->hasCsEntry = true;
    entry->nIndirects = 0;
    CsList_Append(&csp->directFifo, entry);
    ZF_LOGD("%p PutDirect(%p, pcc=%p) cs=%p count=%" PRIu32 " insert", cs, npkt,
            pccEntry, entry, csp->directFifo.count);
  }
  entry->data = npkt;
  entry->freshUntil =
    pkt->timestamp + TscDuration_FromMillis(data->freshnessPeriod);
  return true;
}

static CsEntry*
__Cs_InsertDirect(Cs* cs, Packet* npkt, PInterest* interest)
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
  if (likely(__Cs_PutDirect(cs, npkt, pccEntry))) {
    return PccEntry_GetCsEntry(pccEntry);
  }
  return NULL;
}

static bool
__Cs_PutIndirect(Cs* cs, CsEntry* direct, PccEntry* pccEntry)
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
    CsList_MoveToLast(&csp->indirectFifo, entry);
    ZF_LOGD("%p PutIndirect(%p, pcc=%p) cs=%p count=%" PRIu32 " refresh", cs,
            direct, pccEntry, entry, csp->indirectFifo.count);
  } else {
    // insert indirect entry
    pccEntry->hasCsEntry = true;
    entry->nIndirects = 0;
    CsList_Append(&csp->indirectFifo, entry);
    ZF_LOGD("%p PutIndirect(%p, pcc=%p) cs=%p count=%" PRIu32 " insert", cs,
            direct, pccEntry, entry, csp->indirectFifo.count);
  }

  if (likely(CsEntry_Assoc(entry, direct))) {
    return true;
  }

  ZF_LOGD("^ drop=indirect-assoc-err");
  CsList_Remove(&csp->indirectFifo, entry);
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
    direct = __Cs_InsertDirect(cs, npkt, interest);
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
    bool ok = __Cs_PutDirect(cs, npkt, pccEntry);
    assert(ok);
  } else {
    // put indirect CS entry at pccEntry
    __Cs_PutIndirect(cs, direct, pccEntry);
  }

  // evict if over capacity
  Cs_Evict(cs);
}

bool
__Cs_MatchInterest(Cs* cs, PccEntry* pccEntry, Packet* interestNpkt)
{
  assert(pccEntry->hasCsEntry);
  CsEntry* csEntry = PccEntry_GetCsEntry(pccEntry);
  PInterest* interest = Packet_GetInterestHdr(interestNpkt);
  Packet* dataNpkt = CsEntry_GetData(csEntry);
  assert(dataNpkt != NULL);
  PData* data = Packet_GetDataHdr(dataNpkt);

  bool violateCanBePrefix =
    !interest->canBePrefix && interest->name.p.nComps < data->name.p.nComps;
  bool violateMustBeFresh =
    interest->mustBeFresh &&
    !CsEntry_IsFresh(csEntry, Packet_ToMbuf(interestNpkt)->timestamp);

  if (likely(!violateCanBePrefix && !violateMustBeFresh)) {
    return true;
  }

  if (unlikely(violateCanBePrefix && !interest->mustBeFresh)) {
    // erase CS entry to make room for pitEntry0
    ZF_LOGD("%p MatchInterest(%p) erase-conflict-PIT cs=%p", cs, pccEntry,
            csEntry);
    __Cs_RawErase(cs, csEntry);
  }
  return false;
}

void
__Cs_RawErase(Cs* cs, CsEntry* entry)
{
  CsPriv* csp = Cs_GetPriv(cs);
  PccEntry* pccEntry = PccEntry_FromCsEntry(entry);

  if (CsEntry_IsDirect(entry)) {
    // TODO bulk-erase PccEntry containing indirect entries
    for (int i = 0; i < entry->nIndirects; ++i) {
      Cs_Erase(cs, entry->indirect[i]);
    }
    CsList_Remove(&csp->directFifo, entry);
  } else {
    CsList_Remove(&csp->indirectFifo, entry);
  }

  CsEntry_Finalize(entry);
  pccEntry->hasCsEntry = false;
}

void
Cs_Erase(Cs* cs, CsEntry* entry)
{
  PccEntry* pccEntry = PccEntry_FromCsEntry(entry);

  __Cs_RawErase(cs, entry);

  if (likely(!pccEntry->hasPitEntry1)) {
    Pcct_Erase(Cs_ToPcct(cs), pccEntry);
  }

  ZF_LOGD("%p Erase(%p) pcc=%p", cs, entry, pccEntry);
}

typedef struct CsEvictContext
{
  Cs* cs;
  PccEntry* pccErase[CS_EVICT_BULK];
  uint32_t nPccErase;
} CsEvictContext;

static void
CsEvictContext_Add(CsEvictContext* ctx, CsEntry* entry)
{
  CsEntry_Finalize(entry);

  PccEntry* pccEntry = PccEntry_FromCsEntry(entry);
  if (likely(!pccEntry->hasPitEntry1)) {
    ctx->pccErase[ctx->nPccErase++] = pccEntry;
    ZF_LOGD("^ cs=%p pcc=%p(erase)", entry, pccEntry);
  } else {
    pccEntry->hasCsEntry = false;
    ZF_LOGD("^ cs=%p pcc=%p(retain)", entry, pccEntry);
  }
}

static void
CsEvictContext_AddIndirect(void* ctx0, CsEntry* entry)
{
  CsEvictContext* ctx = (CsEvictContext*)ctx0;
  assert(!CsEntry_IsDirect(entry));
  CsEvictContext_Add(ctx, entry);
}

static void
CsEvictContext_AddDirect(void* ctx0, CsEntry* entry)
{
  CsEvictContext* ctx = (CsEvictContext*)ctx0;
  assert(CsEntry_IsDirect(entry));
  // TODO bulk-erase PccEntry containing indirect entries
  for (int i = 0; i < entry->nIndirects; ++i) {
    Cs_Erase(ctx->cs, entry->indirect[i]);
  }
  CsEvictContext_Add(ctx, entry);
}

static bool
CsEvictContext_Finish(CsEvictContext* ctx)
{
  Pcct_EraseBulk(Cs_ToPcct(ctx->cs), ctx->pccErase, ctx->nPccErase);
}

static void
__Cs_EvictBulk(Cs* cs, CsList* csl, const char* cslName, CsList_EvictCb evictCb)
{
  CsEvictContext ctx = { 0 };
  ctx.cs = cs;
  ZF_LOGD("%p Evict(%s) count=%" PRIu32, cs, cslName, csl->count);
  CsList_EvictBulk(csl, CS_EVICT_BULK, evictCb, &ctx);
  CsEvictContext_Finish(&ctx);
  ZF_LOGD("^ end-count=%" PRIu32, csl->count);
}

static void
Cs_Evict(Cs* cs)
{
  CsPriv* csp = Cs_GetPriv(cs);
  while (unlikely(csp->indirectFifo.capacity <= csp->indirectFifo.count)) {
    __Cs_EvictBulk(cs, &csp->indirectFifo, "indirect",
                   CsEvictContext_AddIndirect);
  }
  while (unlikely(csp->directFifo.capacity <= csp->directFifo.count)) {
    __Cs_EvictBulk(cs, &csp->directFifo, "direct", CsEvictContext_AddDirect);
  }
}
