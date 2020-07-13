#ifndef NDN_DPDK_PCCT_CS_LIST_H
#define NDN_DPDK_PCCT_CS_LIST_H

/** @file */

#include "cs-entry.h"

__attribute__((nonnull)) void
CsList_Init(CsList* csl);

/** @brief Append an entry to back of list. */
__attribute__((nonnull)) void
CsList_Append(CsList* csl, CsEntry* entry);

/** @brief Remove an entry from the list. */
__attribute__((nonnull)) void
CsList_Remove(CsList* csl, CsEntry* entry);

/** @brief Access the front entry of list. */
__attribute__((nonnull)) static inline CsEntry*
CsList_GetFront(CsList* csl)
{
  assert(csl->count > 0);
  return (CsEntry*)csl->next;
}

/** @brief Move an entry to back of list. */
__attribute__((nonnull)) void
CsList_MoveToLast(CsList* csl, CsEntry* entry);

typedef void (*CsList_EvictCb)(void* arg, CsEntry* entry);

/**
 * @brief Evict up to @p max entries from front of list.
 * @param cb callback to erase an entry; the callback must not invoke CsList_Remove.
 */
__attribute__((nonnull(1))) uint32_t
CsList_EvictBulk(CsList* csl, uint32_t max, CsList_EvictCb cb, void* cbarg);

#endif // NDN_DPDK_PCCT_CS_LIST_H
