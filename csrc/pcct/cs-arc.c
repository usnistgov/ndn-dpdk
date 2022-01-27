#include "cs-arc.h"

#include "../core/logger.h"

N_LOG_INIT(CsArc);

const ptrdiff_t CsArc_ListOffsets_[] = {
  [CslDirectT1] = offsetof(CsArc, T1),   [CslDirectB1] = offsetof(CsArc, B1),
  [CslDirectT2] = offsetof(CsArc, T2),   [CslDirectB2] = offsetof(CsArc, B2),
  [CslDirectDel] = offsetof(CsArc, Del),
};

#define CsArc_Move(arc, entry, src, dst)                                                           \
  do {                                                                                             \
    NDNDPDK_ASSERT((entry)->arcList == CslDirect##src);                                            \
    CsList_Remove(&(arc)->src, (entry));                                                           \
    (entry)->arcList = CslDirect##dst;                                                             \
    CsList_Append(&(arc)->dst, (entry));                                                           \
    N_LOGV("^ move=%p from=" #src " to=" #dst, (entry));                                           \
    (arc)->moveCb((arc)->moveCbArg, (entry), CslDirect##src, CslDirect##dst);                      \
  } while (false)

/**
 * @brief Default handler when entry is moved between lists.
 *
 * If an entry is moved from T1/T2 to a non-T1/T2 list, its Data packet is released.
 */
__attribute__((nonnull)) static inline void
CsArc_MoveHandler(__rte_unused void* arg, CsEntry* entry, CsListID src, CsListID dst)
{
  if ((src == CslDirectT1 || src == CslDirectT2) && !(dst == CslDirectT1 || dst == CslDirectT2)) {
    CsEntry_Clear(entry);
  }
}

__attribute__((nonnull)) static inline void
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
  CsList_Init(&arc->Del);

  arc->c = (double)capacity;
  CsArc_c(arc) = capacity;
  CsArc_2c(arc) = 2 * capacity;
  CsArc_SetP(arc, 0.0);

  arc->moveCb = CsArc_MoveHandler;
  arc->moveCbArg = NULL;
}

__attribute__((nonnull)) static void
CsArc_Replace(CsArc* arc, bool isB2)
{
  if (isB2 ? arc->T1.count >= CsArc_p1(arc) : arc->T1.count > CsArc_p(arc)) {
    CsEntry* moving = CsList_GetFront(&arc->T1);
    CsArc_Move(arc, moving, T1, B1);
  } else {
    CsEntry* moving = CsList_GetFront(&arc->T2);
    CsArc_Move(arc, moving, T2, B2);
  }
}

__attribute__((nonnull)) static void
CsArc_AddB1(CsArc* arc, CsEntry* entry)
{
  double delta1 = 1.0;
  if (arc->B1.count < arc->B2.count) {
    delta1 = (double)arc->B2.count / (double)arc->B1.count;
  }
  CsArc_SetP(arc, RTE_MIN(arc->p + delta1, arc->c));
  N_LOGD("Add arc=%p cs-entry=%p found-in=B1 p=%0.3f", arc, entry, arc->p);
  CsArc_Replace(arc, false);
  CsArc_Move(arc, entry, B1, T2);
}

__attribute__((nonnull)) static void
CsArc_AddB2(CsArc* arc, CsEntry* entry)
{
  double delta2 = 1.0;
  if (arc->B2.count < arc->B1.count) {
    delta2 = (double)arc->B1.count / (double)arc->B2.count;
  }
  CsArc_SetP(arc, RTE_MAX(arc->p - delta2, 0.0));
  N_LOGD("Add arc=%p cs-entry=%p found-in=B2 p=%0.3f", arc, entry, arc->p);
  CsArc_Replace(arc, true);
  CsArc_Move(arc, entry, B2, T2);
}

__attribute__((nonnull)) static void
CsArc_AddNew(CsArc* arc, CsEntry* entry)
{
  N_LOGD("Add arc=%p cs-entry=%p found-in=NEW append-to=T1", arc, entry);
  uint32_t nL1 = arc->T1.count + arc->B1.count;
  if (nL1 == CsArc_c(arc)) {
    if (arc->T1.count < CsArc_c(arc)) {
      N_LOGV("^ evict-from=B1");
      CsEntry* deleting = CsList_GetFront(&arc->B1);
      CsArc_Move(arc, deleting, B1, Del);
      CsArc_Replace(arc, false);
    } else {
      NDNDPDK_ASSERT(arc->B1.count == 0);
      N_LOGV("^ evict-from=T1");
      CsEntry* deleting = CsList_GetFront(&arc->T1);
      CsArc_Move(arc, deleting, T1, Del);
    }
  } else {
    NDNDPDK_ASSERT(nL1 < CsArc_c(arc));
    uint32_t nL1L2 = nL1 + arc->T2.count + arc->B2.count;
    if (nL1L2 >= CsArc_c(arc)) {
      if (nL1L2 == CsArc_2c(arc)) {
        N_LOGV("^ evict-from=B2");
        CsEntry* deleting = CsList_GetFront(&arc->B2);
        CsArc_Move(arc, deleting, B2, Del);
      }
      CsArc_Replace(arc, false);
    }
  }
  entry->arcList = CslDirectT1;
  CsList_Append(&arc->T1, entry);
}

void
CsArc_Add(CsArc* arc, CsEntry* entry)
{
  switch (entry->arcList) {
    case CslDirectT1:
      N_LOGD("Add arc=%p cs-entry=%p found-in=T1", arc, entry);
      CsArc_Move(arc, entry, T1, T2);
      return;
    case CslDirectT2:
      N_LOGD("Add arc=%p cs-entry=%p found-in=T2", arc, entry);
      CsList_MoveToLast(&arc->T2, entry);
      return;
    case CslDirectB1:
      CsArc_AddB1(arc, entry);
      return;
    case CslDirectB2:
      CsArc_AddB2(arc, entry);
      return;
    case CslDirectDel:
      CsList_Remove(&arc->Del, entry);
      entry->arcList = 0;
      // fallthrough
    case 0: // this ensures other case constants are non-zero
    default:
      CsArc_AddNew(arc, entry);
      return;
  }
  NDNDPDK_ASSERT(false);
}

void
CsArc_Remove(CsArc* arc, CsEntry* entry)
{
  N_LOGD("Remove arc=%p cs-entry=%p", arc, entry);
  CsList_Remove(CsArc_GetList(arc, entry->arcList), entry);
  entry->arcList = 0;
}
