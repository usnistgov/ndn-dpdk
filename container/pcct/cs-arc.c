#include "cs-arc.h"

#include "../../core/logger.h"

INIT_ZF_LOG(CsArc);

void
CsArc_Init(CsArc* arc, uint32_t capacity)
{
  CsList_Init(&arc->T1);
  CsList_Init(&arc->B1);
  CsList_Init(&arc->T2);
  CsList_Init(&arc->B2);
  CsList_Init(&arc->DEL);

  arc->c = (double)capacity;
  arc->p = 0.0;
  arc->B1.capacity = capacity;
  arc->B2.capacity = 2 * capacity;
}

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
      assert(false);
      return NULL;
  }
}

#define CsArc_Move(arc, entry, src, dst)                                       \
  do {                                                                         \
    assert((entry)->arcList == CSL_ARC_##src);                                 \
    CsList_Remove(&(arc)->src, (entry));                                       \
    (entry)->arcList = CSL_ARC_##dst;                                          \
    CsList_Append(&(arc)->dst, (entry));                                       \
    ZF_LOGV("^ move=%p from=%s to=%s", (entry), #src, #dst);                   \
  } while (false)

static void
CsArc_Replace(CsArc* arc, bool isB2)
{
  CsEntry* moving = NULL;
  if (isB2 ? (arc->T1.count > 0 && arc->T1.count >= arc->T1.capacity)
           : arc->T1.count > arc->T1.capacity) {
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
  if (arc->B1.count >= arc->B2.count) {
    arc->p += 1.0;
  } else {
    arc->p += (double)arc->B2.count / (double)arc->B1.count;
  }
  arc->p = RTE_MIN(arc->p, arc->c);
  arc->T1.capacity = trunc(arc->p);
  ZF_LOGD("%p Add(%p) found-in=B1 p=%0.3f", arc, entry, arc->p);
  CsArc_Replace(arc, false);
  CsArc_Move(arc, entry, B1, T2);
}

static void
CsArc_AddB2(CsArc* arc, CsEntry* entry)
{
  if (arc->B2.count >= arc->B1.count) {
    arc->p -= 1.0;
  } else {
    arc->p -= (double)arc->B1.count / (double)arc->B2.count;
  }
  arc->p = RTE_MAX(arc->p, 0.0);
  arc->T1.capacity = trunc(arc->p);
  ZF_LOGD("%p Add(%p) found-in=B2 p=%0.3f", arc, entry, arc->p);
  CsArc_Replace(arc, true);
  CsArc_Move(arc, entry, B2, T2);
}

static void
CsArc_AddNew(CsArc* arc, CsEntry* entry)
{
  ZF_LOGD("%p Add(%p) found-in=NEW append-to=T1", arc, entry);
  uint32_t nL1 = arc->T1.count + arc->B1.count;
  assert(nL1 <= arc->B1.capacity);
  if (nL1 == arc->B1.capacity) {
    if (arc->T1.count < arc->B1.capacity) {
      ZF_LOGV("^ evict-from=B1");
      CsEntry* deleting = CsList_GetFront(&arc->B1);
      CsArc_Move(arc, deleting, B1, DEL);
      CsArc_Replace(arc, false);
    } else {
      ZF_LOGV("^ evict-from=T1");
      CsEntry* deleting = CsList_GetFront(&arc->T1);
      CsEntry_ClearData(deleting);
      CsArc_Move(arc, deleting, T1, DEL);
    }
  } else {
    uint32_t nL1L2 = nL1 + arc->T2.count + arc->B2.count;
    if (nL1L2 >= arc->B1.capacity) {
      if (nL1L2 == arc->B2.capacity) {
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
  assert(false);
}

void
CsArc_Remove(CsArc* arc, CsEntry* entry)
{
  ZF_LOGD("%p Remove(%p)", arc, entry);
  switch (entry->arcList) {
    case CSL_ARC_T1:
      CsList_Remove(&arc->T1, entry);
      break;
    case CSL_ARC_T2:
      CsList_Remove(&arc->T2, entry);
      break;
    case CSL_ARC_B1:
      CsList_Remove(&arc->B1, entry);
      break;
    case CSL_ARC_B2:
      CsList_Remove(&arc->B2, entry);
      break;
    case CSL_ARC_DEL:
      CsList_Remove(&arc->DEL, entry);
      break;
    default:
      assert(false);
  }
  entry->arcList = CSL_ARC_NONE;
}
