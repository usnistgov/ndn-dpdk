#ifndef NDNDPDK_PCCT_CS_LIST_H
#define NDNDPDK_PCCT_CS_LIST_H

/** @file */

#include "cs-entry.h"

__attribute__((nonnull)) void
CsList_Init(CsList* csl);

static __rte_always_inline void
CsList_AppendNode_(CsList* csl, CsNode* node)
{
  CsNode* last = csl->prev;
  node->prev = last;
  node->next = (CsNode*)csl;
  last->next = node;
  csl->prev = node;
}

static __rte_always_inline void
CsList_RemoveNode_(CsList* csl, CsNode* node)
{
  CsNode* prev = node->prev;
  CsNode* next = node->next;
  NDNDPDK_ASSERT(prev->next == node);
  NDNDPDK_ASSERT(next->prev == node);
  prev->next = next;
  next->prev = prev;
}

/** @brief Append an entry to back of list. */
__attribute__((nonnull)) static __rte_always_inline void
CsList_Append(CsList* csl, CsEntry* entry)
{
  CsList_AppendNode_(csl, (CsNode*)entry);
  ++csl->count;
}

/** @brief Remove an entry from the list. */
__attribute__((nonnull)) static __rte_always_inline void
CsList_Remove(CsList* csl, CsEntry* entry)
{
  NDNDPDK_ASSERT(csl->count > 0);
  CsList_RemoveNode_(csl, (CsNode*)entry);
  --csl->count;
}

/** @brief Access the front entry of list. */
__attribute__((nonnull, returns_nonnull)) static __rte_always_inline CsEntry*
CsList_GetFront(CsList* csl)
{
  NDNDPDK_ASSERT(csl->count > 0);
  return (CsEntry*)csl->next;
}

/** @brief Move an entry to back of list. */
__attribute__((nonnull)) static __rte_always_inline void
CsList_MoveToLast(CsList* csl, CsEntry* entry)
{
  CsList_RemoveNode_(csl, (CsNode*)entry);
  CsList_AppendNode_(csl, (CsNode*)entry);
}

typedef void (*CsList_EvictCb)(void* arg, CsEntry* entry);

/**
 * @brief Evict up to @p max entries from front of list.
 * @param cb callback to erase an entry; the callback must not invoke CsList_Remove.
 */
__attribute__((nonnull(1))) uint32_t
CsList_EvictBulk(CsList* csl, uint32_t max, CsList_EvictCb cb, void* arg);

#endif // NDNDPDK_PCCT_CS_LIST_H
