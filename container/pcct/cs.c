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
Cs_Insert(Cs* cs, Packet* npkt, PitEntry* pitEntry)
{
  CsPriv* csp = Cs_GetPriv(cs);

  // TODO check Data has exact name as the PIT entry

  // XXX will crash if pitEntry is for MustBeFresh=1
  PccEntry* pccEntry = __Pit_RawErase0(Pit_FromPcct(Cs_ToPcct(cs)), pitEntry);

  pccEntry->hasCsEntry = true;
  CsEntry* entry = &pccEntry->csEntry;
  entry->data = npkt;

  CsPriv_AppendEntry(csp, entry);

  // TODO evict if needed

  ZF_LOGD("%p Insert(%p, pit=%p) pcc=%p cs=%p", cs, npkt, pitEntry, pccEntry,
          entry);
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
