#include "cs-list.h"

static_assert(offsetof(CsNode, prev) == offsetof(CsEntry, prev), "");
static_assert(offsetof(CsNode, next) == offsetof(CsEntry, next), "");
static_assert(offsetof(CsNode, prev) == offsetof(CsList, prev), "");
static_assert(offsetof(CsNode, next) == offsetof(CsList, next), "");

void
CsList_Init(CsList* csl) {
  csl->prev = csl->next = (CsNode*)csl;
  csl->count = 0;
  csl->capacity = 0;
}

uint32_t
CsList_EvictBulk(CsList* csl, uint32_t max, CsList_EvictCb cb, uintptr_t ctx) {
  uint32_t nErase = RTE_MIN(max, csl->count);
  CsNode* node = csl->next;

  for (uint32_t i = 0; i < nErase; ++i) {
    NDNDPDK_ASSERT(node != (CsNode*)csl);
    CsEntry* entry = (CsEntry*)node;
    node = node->next;
    cb(entry, ctx);
  }

  node->prev = (CsNode*)csl;
  csl->next = node;
  csl->count -= nErase;

  return nErase;
}
