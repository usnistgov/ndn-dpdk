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

static __rte_always_inline void
CsList_AppendNode(CsList* csl, CsNode* node)
{
  CsNode* last = csl->prev;
  node->prev = last;
  node->next = (CsNode*)csl;
  last->next = node;
  csl->prev = node;
}

static __rte_always_inline void
CsList_RemoveNode(CsList* csl, CsNode* node)
{
  CsNode* prev = node->prev;
  CsNode* next = node->next;
  NDNDPDK_ASSERT(prev->next == node);
  NDNDPDK_ASSERT(next->prev == node);
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
  NDNDPDK_ASSERT(csl->count > 0);
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

  for (uint32_t i = 0; i < nErase; ++i) {
    NDNDPDK_ASSERT(node != (CsNode*)csl);
    CsEntry* entry = (CsEntry*)node;
    node = node->next;
    cb(cbarg, entry);
  }

  node->prev = (CsNode*)csl;
  csl->next = node;
  csl->count -= nErase;

  return nErase;
}
