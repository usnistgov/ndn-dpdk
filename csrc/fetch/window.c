#include "window.h"

void
FetchWindow_Init(FetchWindow* win, uint32_t capacity, int numaSocket)
{
  // txTime has room for RTT up to 300s
  NDNDPDK_ASSERT(TscDuration_FromMillis(300000) < (TscDuration)FetchSegTxTimeMask);

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
FetchWindow_Reset(FetchWindow* win, uint64_t firstSegNum)
{
  win->loSegNum = firstSegNum;
  win->hiSegNum = firstSegNum;
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
