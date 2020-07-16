#ifndef NDNDPDK_PCCT_CS_ARC_H
#define NDNDPDK_PCCT_CS_ARC_H

/** @file */

#include "cs-list.h"

__attribute__((nonnull)) void
CsArc_Init(CsArc* arc, uint32_t capacity);

__attribute__((nonnull)) CsList*
CsArc_GetList(CsArc* arc, CsArcListId cslId);

__attribute__((nonnull)) static __rte_always_inline uint32_t
CsArc_GetCapacity(const CsArc* arc)
{
  return arc->B1.capacity;
}

static __rte_always_inline uint32_t
CsArc_CountEntries(CsArc* arc)
{
  return arc->T1.count + arc->T2.count;
}

__attribute__((nonnull)) void
CsArc_Add(CsArc* arc, CsEntry* entry);

__attribute__((nonnull)) void
CsArc_Remove(CsArc* arc, CsEntry* entry);

#endif // NDNDPDK_PCCT_CS_ARC_H
