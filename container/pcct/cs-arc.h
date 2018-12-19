#ifndef NDN_DPDK_CONTAINER_PCCT_CS_ARC_H
#define NDN_DPDK_CONTAINER_PCCT_CS_ARC_H

/// \file

#include "cs-list.h"

void CsArc_Init(CsArc* arc, uint32_t capacity);

CsList* CsArc_GetList(CsArc* arc, CsArcListId cslId);

static uint32_t
CsArc_GetCapacity(CsArc* arc)
{
  return arc->B1.capacity;
}

static uint32_t
CsArc_CountEntries(CsArc* arc)
{
  return arc->T1.count + arc->T2.count;
}

void CsArc_Add(CsArc* arc, CsEntry* entry);

void CsArc_Remove(CsArc* arc, CsEntry* entry);

#endif // NDN_DPDK_CONTAINER_PCCT_CS_ARC_H
