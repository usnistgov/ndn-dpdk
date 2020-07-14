#include "cs-arc.h"

#include "../core/logger.h"

INIT_ZF_LOG(CsArc);

CsList*
CsArc_GetList(CsArc* arc, CsArcListId cslId)
{
  switch (cslId) {
    case CSL_ARC_T1:
      return &arc->T1;
    case CSL_ARC_B1:
      return &arc->B1;
    case CSL_ARC_T2:
      return &arc->T2;
    case CSL_ARC_B2:
      return &arc->B2;
    case CSL_ARC_DEL:
      return &arc->DEL;
    default:
      NDNDPDK_ASSERT(false);
      return NULL;
  }
}

#define CsArc_c(arc) ((arc)->B1.capacity)
#define CsArc_2c(arc) ((arc)->B2.capacity)
#define CsArc_p(arc) ((arc)->T1.capacity)
#define CsArc_p1(arc) ((arc)->T2.capacity)

#define CsArc_Move(arc, entry, src, dst)                                                           \
  do {                                                                                             \
    NDNDPDK_ASSERT((entry)->arcList == CSL_ARC_##src);                                             \
    CsList_Remove(&(arc)->src, (entry));                                                           \
    (entry)->arcList = CSL_ARC_##dst;                                                              \
    CsList_Append(&(arc)->dst, (entry));                                                           \
    ZF_LOGV("^ move=%p from=" #src " to=" #dst, (entry));                                          \
  } while (false)

static inline void
CsArc_SetP(CsArc* arc, double p)
{
  arc->p = p;
  CsArc_p(arc) = (uint32_t)p;
  CsArc_p1(arc) = RTE_MAX(CsArc_p(arc), 1);
}

void
CsArc_Init(CsArc* arc, uint32_t capacity)
{
  CsList_Init(&arc->T1);
  CsList_Init(&arc->B1);
  CsList_Init(&arc->T2);
  CsList_Init(&arc->B2);
  CsList_Init(&arc->DEL);

  arc->c = (double)capacity;
  CsArc_c(arc) = capacity;
  CsArc_2c(arc) = 2 * capacity;
  CsArc_SetP(arc, 0.0);
}

static void
CsArc_Replace(CsArc* arc, bool isB2)
{
  CsEntry* moving = NULL;
  if (isB2 ? arc->T1.count >= CsArc_p1(arc) : arc->T1.count > CsArc_p(arc)) {
    moving = CsList_GetFront(&arc->T1);
    CsArc_Move(arc, moving, T1, B1);
  } else {
    moving = CsList_GetFront(&arc->T2);
    CsArc_Move(arc, moving, T2, B2);
  }
  CsEntry_ClearData(moving);
}

static void
CsArc_AddB1(CsArc* arc, CsEntry* entry)
{
  double delta1 = 1.0;
  if (arc->B1.count < arc->B2.count) {
    delta1 = (double)arc->B2.count / (double)arc->B1.count;
  }
  CsArc_SetP(arc, RTE_MIN(arc->p + delta1, arc->c));
  ZF_LOGD("%p Add(%p) found-in=B1 p=%0.3f", arc, entry, arc->p);
  CsArc_Replace(arc, false);
  CsArc_Move(arc, entry, B1, T2);
}

static void
CsArc_AddB2(CsArc* arc, CsEntry* entry)
{
  double delta2 = 1.0;
  if (arc->B2.count < arc->B1.count) {
    delta2 = (double)arc->B1.count / (double)arc->B2.count;
  }
  CsArc_SetP(arc, RTE_MAX(arc->p - delta2, 0.0));
  ZF_LOGD("%p Add(%p) found-in=B2 p=%0.3f", arc, entry, arc->p);
  CsArc_Replace(arc, true);
  CsArc_Move(arc, entry, B2, T2);
}

static void
CsArc_AddNew(CsArc* arc, CsEntry* entry)
{
  ZF_LOGD("%p Add(%p) found-in=NEW append-to=T1", arc, entry);
  uint32_t nL1 = arc->T1.count + arc->B1.count;
  if (nL1 == CsArc_c(arc)) {
    if (arc->T1.count < CsArc_c(arc)) {
      ZF_LOGV("^ evict-from=B1");
      CsEntry* deleting = CsList_GetFront(&arc->B1);
      CsArc_Move(arc, deleting, B1, DEL);
      CsArc_Replace(arc, false);
    } else {
      NDNDPDK_ASSERT(arc->B1.count == 0);
      ZF_LOGV("^ evict-from=T1");
      CsEntry* deleting = CsList_GetFront(&arc->T1);
      CsEntry_ClearData(deleting);
      CsArc_Move(arc, deleting, T1, DEL);
    }
  } else {
    NDNDPDK_ASSERT(nL1 < CsArc_c(arc));
    uint32_t nL1L2 = nL1 + arc->T2.count + arc->B2.count;
    if (nL1L2 >= CsArc_c(arc)) {
      if (nL1L2 == CsArc_2c(arc)) {
        ZF_LOGV("^ evict-from=B2");
        CsEntry* deleting = CsList_GetFront(&arc->B2);
        CsArc_Move(arc, deleting, B2, DEL);
      }
      CsArc_Replace(arc, false);
    }
  }
  entry->arcList = CSL_ARC_T1;
  CsList_Append(&arc->T1, entry);
}

void
CsArc_Add(CsArc* arc, CsEntry* entry)
{
  switch (entry->arcList) {
    case CSL_ARC_T1:
      ZF_LOGD("%p Add(%p) found-in=T1", arc, entry);
      CsArc_Move(arc, entry, T1, T2);
      return;
    case CSL_ARC_T2:
      ZF_LOGD("%p Add(%p) found-in=T2", arc, entry);
      CsList_MoveToLast(&arc->T2, entry);
      return;
    case CSL_ARC_B1:
      CsArc_AddB1(arc, entry);
      return;
    case CSL_ARC_B2:
      CsArc_AddB2(arc, entry);
      return;
    case CSL_ARC_DEL:
      CsList_Remove(&arc->DEL, entry);
      entry->arcList = CSL_ARC_NONE;
    // fallthrough
    case CSL_ARC_NONE:
      CsArc_AddNew(arc, entry);
      return;
  }
  NDNDPDK_ASSERT(false);
}

void
CsArc_Remove(CsArc* arc, CsEntry* entry)
{
  ZF_LOGD("%p Remove(%p)", arc, entry);
  CsList_Remove(CsArc_GetList(arc, entry->arcList), entry);
  entry->arcList = CSL_ARC_NONE;
}
