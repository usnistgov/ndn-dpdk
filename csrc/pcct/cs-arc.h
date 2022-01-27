#ifndef NDNDPDK_PCCT_CS_ARC_H
#define NDNDPDK_PCCT_CS_ARC_H

/** @file */

#include "cs-list.h"

extern const ptrdiff_t CsArc_ListOffsets_[];

/**
 * @brief Retrieve a CsList by ID.
 * @param l list identifier, which must exist in CsArc struct.
 */
__attribute__((nonnull, returns_nonnull)) static __rte_always_inline CsList*
CsArc_GetList(CsArc* arc, CsListID l)
{
  return RTE_PTR_ADD(arc, CsArc_ListOffsets_[l]);
}

/**
 * @brief Initialize ARC.
 * @param capacity nominal capacity @c c .
 */
__attribute__((nonnull)) void
CsArc_Init(CsArc* arc, uint32_t capacity);

/** @brief Return nominal capacity @c c . */
__attribute__((nonnull)) static __rte_always_inline uint32_t
CsArc_GetCapacity(const CsArc* arc)
{
  return arc->B1.capacity;
}

/** @brief Return number of in-memory entries. */
static __rte_always_inline uint32_t
CsArc_CountEntries(CsArc* arc)
{
  return arc->T1.count + arc->T2.count;
}

/** @brief Add or refresh an entry. */
__attribute__((nonnull)) void
CsArc_Add(CsArc* arc, CsEntry* entry);

/** @brief Remove an entry. */
__attribute__((nonnull)) void
CsArc_Remove(CsArc* arc, CsEntry* entry);

#endif // NDNDPDK_PCCT_CS_ARC_H
