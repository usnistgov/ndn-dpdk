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

void
Cs_Insert(Cs* cs, Packet* npkt, PitResult pitFound)
{
  CsPriv* csp = Cs_GetPriv(cs);
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  PData* data = Packet_GetDataHdr(npkt);
  PccEntry* pccEntry = __PitResult_GetPccEntry(pitFound);

  // delete PIT entries
  {
    Pit* pit = Pit_FromPcct(Cs_ToPcct(cs));
    __Pit_RawErase01(pit, pccEntry);
  }

  // Data has exact name?
  PInterest* interest = __PitFindResult_GetInterest(pitFound);
  if (unlikely(interest->name.p.nComps != data->name.p.nComps)) {
    // Interest name is a prefix of Data name
    ZF_LOGD("%p Insert(%p, pcc=%p) drop=inexact-name", cs, npkt, pccEntry);
    // TODO insert Data into a new PccEntry
    rte_pktmbuf_free(Packet_ToMbuf(npkt));
    Pcct_Erase(Cs_ToPcct(cs), pccEntry);
    return;
  }

  CsEntry* entry = &pccEntry->csEntry;
  if (unlikely(pccEntry->hasCsEntry)) {
    // refresh CS entry
    rte_pktmbuf_free(Packet_ToMbuf(entry->data));
    CsPriv_MoveEntryToLast(csp, entry);
    ZF_LOGD("%p Insert(%p, pcc=%p) cs=%p count=%" PRIu32 " refresh", cs, npkt,
            pccEntry, entry, csp->nEntries);
  } else {
    // insert CS entry
    pccEntry->hasCsEntry = true;
    CsPriv_AppendEntry(csp, entry);
    ZF_LOGD("%p Insert(%p, pcc=%p) cs=%p count=%" PRIu32 " insert", cs, npkt,
            pccEntry, entry, csp->nEntries);
  }
  entry->data = npkt;
  entry->freshUntil =
    pkt->timestamp + TscDuration_FromMillis(data->freshnessPeriod);

  // evict if over capacity
  if (unlikely(csp->nEntries > csp->capacity)) {
    Cs_EvictBulk(cs);
  }
}

void
Cs_Erase(Cs* cs, CsEntry* entry)
{
  CsPriv* csp = Cs_GetPriv(cs);
  PccEntry* pccEntry = PccEntry_FromCsEntry(entry);

  CsPriv_RemoveEntry(csp, entry);
  CsEntry_Finalize(entry);

  if (likely(!pccEntry->hasPitEntry1)) {
    Pcct_Erase(Cs_ToPcct(cs), pccEntry);
  } else {
    pccEntry->hasCsEntry = false;
  }

  ZF_LOGD("%p Erase(%p) pcc=%p", cs, entry, pccEntry);
}
