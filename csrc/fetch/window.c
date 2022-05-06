#include "window.h"

void
FetchWindow_Init(FetchWindow* win, uint32_t capacity, int numaSocket)
{
  NDNDPDK_ASSERT(rte_is_power_of_2(capacity));

  size_t sizeofArray = sizeof(FetchSeg) * capacity;
  size_t sizeofDeletedBitmap = sizeof(uint64_t) * SPDK_CEIL_DIV(capacity, 64);
  *win = (FetchWindow){
    .array = rte_zmalloc_socket("FetchWindow", sizeofArray + sizeofDeletedBitmap,
                                RTE_CACHE_LINE_SIZE, numaSocket),
    .capacityMask = capacity - 1,
  };

  NDNDPDK_ASSERT(win->array != NULL);
  win->deleted = RTE_PTR_ADD(win->array, sizeofArray);
}

void
FetchWindow_Free(FetchWindow* win)
{
  rte_free(win->array);
  NULLize(win->array);
  NULLize(win->deleted);
}

void
FetchWindow_Advance_(FetchWindow* win)
{
  while (win->loSegNum < win->hiSegNum) {
    FetchSeg* seg = NULL;
    uint64_t* deletedSlab = NULL;
    uint64_t deletedBit = 0;
    FetchWindow_Pos_(win, win->loSegNum, &seg, &deletedSlab, &deletedBit);
    if (unlikely((*deletedSlab & deletedBit) == 0)) {
      break;
    }
    if (*deletedSlab == UINT64_MAX) {
      win->loSegNum = RTE_MIN(win->hiSegNum, RTE_ALIGN_CEIL(win->loSegNum + 1, 64));
      continue;
    }
    ++win->loSegNum;
  }
}
