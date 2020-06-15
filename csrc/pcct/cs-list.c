#include "cs-list.h"

struct CsNode
{
  CsNode* prev;
  CsNode* next;
};

static_assert(offsetof(CsNode, prev) == offsetof(CsEntry, prev), "");
static_assert(offsetof(CsNode, next) == offsetof(CsEntry, next), "");
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
  CsList_AppendNode(csl, (CsNode*)entry);
  ++csl->count;
}

void
CsList_Remove(CsList* csl, CsEntry* entry)
{
  assert(csl->count > 0);
  CsList_RemoveNode(csl, (CsNode*)entry);
  --csl->count;
}

void
CsList_MoveToLast(CsList* csl, CsEntry* entry)
{
  CsList_RemoveNode(csl, (CsNode*)entry);
  CsList_AppendNode(csl, (CsNode*)entry);
}

uint32_t
CsList_EvictBulk(CsList* csl, uint32_t max, CsList_EvictCb cb, void* cbarg)
{
  uint32_t nErase = RTE_MIN(max, csl->count);
  CsNode* node = csl->next;

  for (int i = 0; i < nErase; ++i) {
    assert(node != (CsNode*)csl);
    CsEntry* entry = (CsEntry*)node;
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
    case CSL_MD_T1:
      return "MD.T1";
    case CSL_MD_B1:
      return "MD.B1";
    case CSL_MD_T2:
      return "MD.T2";
    case CSL_MD_B2:
      return "MD.B2";
    case CSL_MD_DEL:
      return "MD.DEL";
    case CSL_MI:
      return "MI";
  }
  return "(unknown)";
}
