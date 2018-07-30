#include "cs.h"
#include "pit.h"

#include "../../core/logger.h"

INIT_ZF_LOG(Cs);

static void
CsPriv_AppendNode(CsPriv* csp, CsNode* node)
{
  CsNode* head = &csp->head;
  CsNode* last = head->prev;
  node->prev = last;
  node->next = head;
  last->next = node;
  head->prev = node;
}

static void
CsPriv_RemoveNode(CsPriv* csp, CsNode* node)
{
  CsNode* prev = node->prev;
  CsNode* next = node->next;
  assert(prev->next == node);
  assert(next->prev == node);
  prev->next = next;
  next->prev = prev;
}

static void
CsPriv_AppendEntry(CsPriv* csp, CsEntry* entry)
{
  CsPriv_AppendNode(csp, &entry->node);
  ++csp->nEntries;
}

static void
CsPriv_RemoveEntry(CsPriv* csp, CsEntry* entry)
{
  CsPriv_RemoveNode(csp, &entry->node);
  assert(csp->nEntries > 0);
  --csp->nEntries;
}

static void
CsPriv_MoveEntryToLast(CsPriv* csp, CsEntry* entry)
{
  CsNode* node = &entry->node;
  CsPriv_RemoveNode(csp, node);
  CsPriv_AppendNode(csp, node);
}

static void
Cs_EvictBulk(Cs* cs)
{
  CsPriv* csp = Cs_GetPriv(cs);
  assert(csp->nEntries >= CS_EVICT_BULK);

  ZF_LOGD("%p EvictBulk() count=%" PRIu32, cs, csp->nEntries);

  CsNode* head = &csp->head;
  CsNode* node = head->next;

  PccEntry* pccErase[CS_EVICT_BULK];
  uint32_t nPccErase = 0;

  for (int i = 0; i < CS_EVICT_BULK; ++i) {
    assert(node != head);
    CsEntry* entry = container_of(node, CsEntry, node);
    node = node->next;
    CsEntry_Finalize(entry);

    PccEntry* pccEntry = PccEntry_FromCsEntry(entry);
    if (likely(!pccEntry->hasPitEntry1)) {
      pccErase[nPccErase++] = pccEntry;
    } else {
      pccEntry->hasCsEntry = false;
    }
    ZF_LOGD("^ cs=%p pcc=%p%s", entry, pccEntry,
            pccEntry->hasPitEntry1 ? "(retain)" : "(erase)");
  }

  node->prev = head;
  head->next = node;
  csp->nEntries -= CS_EVICT_BULK;
  ZF_LOGD("^ end-count=%" PRIu32, csp->nEntries);
  Pcct_EraseBulk(Cs_ToPcct(cs), pccErase, nPccErase);
}

void
Cs_Init(Cs* cs, uint32_t capacity)
{
  CsPriv* csp = Cs_GetPriv(cs);
  csp->capacity = RTE_MAX(capacity, CS_EVICT_BULK);
  ZF_LOGI("%p Init() priv=%p capacity=%" PRIu32, cs, csp, csp->capacity);

  CsNode* head = &csp->head;
  head->prev = head->next = head;
}

void
Cs_SetCapacity(Cs* cs, uint32_t capacity)
{
  CsPriv* csp = Cs_GetPriv(cs);
  csp->capacity = RTE_MAX(capacity, CS_EVICT_BULK);
  ZF_LOGI("%p SetCapacity(%" PRIu32 ")", cs, csp->capacity);

  while (likely(csp->nEntries >= csp->capacity)) {
    Cs_EvictBulk(cs);
  }
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
    CsPriv_MoveEntryToLast(csp, entry);
    ZF_LOGD("%p PutDirect(%p, pcc=%p) cs=%p count=%" PRIu32 " refresh", cs,
            npkt, pccEntry, entry, csp->nEntries);
  } else if (unlikely(pccEntry->hasPitEntry0)) {
    ZF_LOGD("%p PutDirect(%p, pcc=%p) drop=has-pit0", cs, npkt, pccEntry);
    return false;
  } else {
    // insert direct entry
    pccEntry->hasCsEntry = true;
    entry->nIndirects = 0;
    CsPriv_AppendEntry(csp, entry);
    ZF_LOGD("%p PutDirect(%p, pcc=%p) cs=%p count=%" PRIu32 " insert", cs, npkt,
            pccEntry, entry, csp->nEntries);
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
    CsPriv_MoveEntryToLast(csp, entry);
    ZF_LOGD("%p PutIndirect(%p, pcc=%p) cs=%p count=%" PRIu32 " refresh", cs,
            direct, pccEntry, entry, csp->nEntries);
  } else {
    // insert indirect entry
    pccEntry->hasCsEntry = true;
    entry->nIndirects = 0;
    CsPriv_AppendEntry(csp, entry);
    ZF_LOGD("%p PutIndirect(%p, pcc=%p) cs=%p count=%" PRIu32 " insert", cs,
            direct, pccEntry, entry, csp->nEntries);
  }

  if (likely(CsEntry_Assoc(entry, direct))) {
    // ensure direct entry is evicted later than indirect entry
    CsPriv_MoveEntryToLast(csp, direct);
    return true;
  }

  ZF_LOGD("^ drop=indirect-assoc-err");
  CsPriv_RemoveEntry(csp, entry);
  pccEntry->hasCsEntry = false;
  Pcct_Erase(Cs_ToPcct(cs), pccEntry);
  return false;
}

void
Cs_Insert(Cs* cs, Packet* npkt, PitResult pitFound)
{
  Pcct* pcct = Cs_ToPcct(cs);
  Pit* pit = Pit_FromPcct(pcct);
  CsPriv* csp = Cs_GetPriv(cs);
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  PData* data = Packet_GetDataHdr(npkt);
  PccEntry* pccEntry = __PitResult_GetPccEntry(pitFound);
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
  if (unlikely(csp->nEntries > csp->capacity)) {
    Cs_EvictBulk(cs);
  }
}

void
__Cs_RawErase(Cs* cs, CsEntry* entry)
{
  CsPriv* csp = Cs_GetPriv(cs);
  PccEntry* pccEntry = PccEntry_FromCsEntry(entry);

  if (CsEntry_IsDirect(entry)) {
    for (int i = 0; i < entry->nIndirects; ++i) {
      Cs_Erase(cs, entry->indirect[i]);
    }
  }

  CsPriv_RemoveEntry(csp, entry);
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
