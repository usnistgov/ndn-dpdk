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

void
Cs_Init(Cs* cs)
{
  CsPriv* csp = Cs_GetPriv(cs);

  CsNode* head = &csp->head;
  head->prev = head->next = head;
}

void
Cs_SetCapacity(Cs* cs, uint32_t capacity)
{
  assert(false); // not implemented
}

void
Cs_Insert(Cs* cs, Packet* npkt, PitResult pitFound)
{
  CsPriv* csp = Cs_GetPriv(cs);
  PData* data = Packet_GetDataHdr(npkt);

  // Data has exact name?
  PInterest* interest = __PitFindResult_GetInterest(pitFound);
  if (unlikely(interest->name.p.nComps != data->name.p.nComps)) {
    // Interest name is a prefix of Data name
    // TODO insert Data into a new PccEntry
    rte_pktmbuf_free(Packet_ToMbuf(npkt));
    return;
  }

  PccEntry* pccEntry = __PitResult_GetPccEntry(pitFound);

  // delete PIT entries
  {
    Pit* pit = Pit_FromPcct(Cs_ToPcct(cs));
    PitEntry* pitEntry0 = PitFindResult_GetPitEntry0(pitFound);
    if (pitEntry0 != NULL) {
      __Pit_RawErase0(pit, pitEntry0);
    }
    PitEntry* pitEntry1 = PitFindResult_GetPitEntry1(pitFound);
    if (pitEntry1 != NULL) {
      __Pit_RawErase1(pit, pitEntry1);
    }
    // TODO optimize this part
  }

  pccEntry->hasCsEntry = true;
  CsEntry* entry = &pccEntry->csEntry;
  entry->data = npkt;

  CsPriv_AppendEntry(csp, entry);

  // TODO evict if needed

  ZF_LOGD("%p Insert(%p, pcc=%p) cs=%p", cs, npkt, pccEntry, entry);
}

void
Cs_Erase(Cs* cs, CsEntry* entry)
{
  CsPriv* csp = Cs_GetPriv(cs);
  PccEntry* pccEntry = PccEntry_FromCsEntry(entry);

  CsPriv_RemoveEntry(csp, entry);
  CsEntry_Finalize(entry);

  pccEntry->hasCsEntry = false;
  if (!pccEntry->hasPitEntry1) {
    Pcct_Erase(Cs_ToPcct(cs), pccEntry);
  }

  ZF_LOGD("%p Erase(%p) pcc=%p", cs, entry, pccEntry);
}
