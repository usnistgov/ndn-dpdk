#include "cs-struct.h"
#include "cs-entry.h"

static_assert(offsetof(CsEntry, node) == 0, "");
static_assert(offsetof(CsNode, prev) == offsetof(CsList, prev), "");
static_assert(offsetof(CsNode, next) == offsetof(CsList, next), "");

void
CsList_Init(CsList* csl)
{
  csl->prev = csl->next = (CsNode*)csl;
  csl->count = 0;
  csl->capacity = 0;
}

static void
CsList_AppendNode(CsList* csl, CsNode* node)
{
  CsNode* last = csl->prev;
  node->prev = last;
  node->next = (CsNode*)csl;
  last->next = node;
  csl->prev = node;
}

static void
CsList_RemoveNode(CsList* csl, CsNode* node)
{
  CsNode* prev = node->prev;
  CsNode* next = node->next;
  assert(prev->next == node);
  assert(next->prev == node);
  prev->next = next;
  next->prev = prev;
}

void
CsList_Append(CsList* csl, CsEntry* entry)
{
  CsList_AppendNode(csl, &entry->node);
  ++csl->count;
}

void
CsList_Remove(CsList* csl, CsEntry* entry)
{
  assert(csl->count > 0);
  CsList_RemoveNode(csl, &entry->node);
  --csl->count;
}

void
CsList_MoveToLast(CsList* csl, CsEntry* entry)
{
  CsNode* node = &entry->node;
  CsList_RemoveNode(csl, node);
  CsList_AppendNode(csl, node);
}

uint32_t
CsList_EvictBulk(CsList* csl, uint32_t max, CsList_EvictCb cb, void* cbarg)
{
  uint32_t nErase = RTE_MIN(max, csl->count);
  CsNode* node = csl->next;

  for (int i = 0; i < nErase; ++i) {
    assert(node != (CsNode*)csl);
    CsEntry* entry = CsNode_AsEntry(node);
    node = node->next;
    cb(cbarg, entry);
  }

  node->prev = (CsNode*)csl;
  csl->next = node;
  csl->count -= nErase;

  return nErase;
}

const char*
CsListId_GetName(CsListId cslId)
{
  switch (cslId) {
    case CSL_MD:
      return "MD";
    case CSL_MI:
      return "MI";
  }
}
