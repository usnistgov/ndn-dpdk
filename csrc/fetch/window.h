#ifndef NDNDPDK_FETCH_WINDOW_H
#define NDNDPDK_FETCH_WINDOW_H

/** @file */

#include "../core/mintmr.h"

/** @brief Per-segment state. */
typedef struct FetchSeg
{
  uint64_t segNum;               ///< segment number
  TscTime txTime;                ///< last Interest tx time
  MinTmr rtoExpiry;              ///< RTO expiration timer
  struct cds_list_head retxNode; ///< retxQ node
  uint16_t nRetx;                ///< number of Interest retx, increment upon TX
} __rte_cache_aligned FetchSeg;

__attribute__((nonnull)) static inline void
FetchSeg_Init(FetchSeg* seg, uint64_t segNum)
{
  seg->segNum = segNum;
  seg->txTime = 0;
  MinTmr_Init(&seg->rtoExpiry);
  CDS_INIT_LIST_HEAD(&seg->retxNode);
  seg->nRetx = 0;
}

/** @brief Window of segment states. */
typedef struct FetchWindow
{
  FetchSeg* array;
  uint64_t* deleted;     ///< deleted flag bitmap
  uint32_t capacityMask; ///< array capacity minus one
  uint64_t loSegNum;     ///< inclusive lower bound of segment numbers
  uint64_t hiSegNum;     ///< exclusive upper bound of segment numbers
} FetchWindow;

/**
 * @brief Initialize FetchWindow.
 * @param capacity maximum distance between lower and upper bounds of segment numbers.
 * @return whether success.
 */
__attribute__((nonnull)) void
FetchWindow_Init(FetchWindow* win, uint32_t capacity, int numaSocket);

/** @brief Deallocated memory. */
__attribute__((nonnull)) void
FetchWindow_Free(FetchWindow* win);

/**
 * @brief Compute position of a segment number.
 * @param segNum segment number, must be within @c [loSegNum,hiSegNum) .
 * @param[out] seg array element.
 * @param[out] deletedSlab deleted bitmap slab position.
 * @param[out] deletedBit deleted bitmap bit within slab.
 */
__attribute__((nonnull)) static __rte_always_inline void
FetchWindow_Pos_(FetchWindow* win, uint64_t segNum, FetchSeg** seg, uint64_t** deletedSlab,
                 uint64_t* deletedBit)
{
  uint64_t pos = segNum & win->capacityMask;
  *seg = &win->array[pos];
  *deletedSlab = &win->deleted[pos >> 6];
  *deletedBit = RTE_BIT64(pos & 0x3F);
}

/** @brief Move loPos and loSegNum after some segment states have been deleted. */
__attribute__((nonnull)) void
FetchWindow_Advance_(FetchWindow* win);

__attribute__((nonnull)) static __rte_always_inline FetchSeg*
FetchWindow_GetOrDelete_(FetchWindow* win, uint64_t segNum, bool isDelete)
{
  if (unlikely(segNum < win->loSegNum || segNum >= win->hiSegNum)) {
    return NULL;
  }

  FetchSeg* seg = NULL;
  uint64_t* deletedSlab = NULL;
  uint64_t deletedBit = 0;
  FetchWindow_Pos_(win, segNum, &seg, &deletedSlab, &deletedBit);
  if (unlikely((*deletedSlab & deletedBit) != 0)) {
    return NULL;
  }

  if (isDelete) {
    *deletedSlab |= deletedBit;
    if (segNum == win->loSegNum) {
      FetchWindow_Advance_(win);
    }
  }

  return seg;
}

/**
 * @brief Retrieve a segment's state.
 * @retval NULL segment is not in the window or has been deleted.
 */
__attribute__((nonnull)) static inline FetchSeg*
FetchWindow_Get(FetchWindow* win, uint64_t segNum)
{
  return FetchWindow_GetOrDelete_(win, segNum, false);
}

/**
 * @brief Create state for the next segment.
 * @retval NULL window has reached its capacity limit.
 */
__attribute__((nonnull)) static inline FetchSeg*
FetchWindow_Append(FetchWindow* win)
{
  uint64_t segNum = win->hiSegNum;
  if (unlikely(segNum - win->loSegNum > win->capacityMask)) {
    return NULL;
  }
  ++win->hiSegNum;

  FetchSeg* seg = NULL;
  uint64_t* deletedSlab = NULL;
  uint64_t deletedBit = 0;
  FetchWindow_Pos_(win, segNum, &seg, &deletedSlab, &deletedBit);
  *deletedSlab &= ~deletedBit;
  FetchSeg_Init(seg, segNum);
  return seg;
}

/** @brief Discard a segment's state. */
__attribute__((nonnull)) static inline void
FetchWindow_Delete(FetchWindow* win, uint64_t segNum)
{
  FetchWindow_GetOrDelete_(win, segNum, true);
}

#endif // NDNDPDK_FETCH_WINDOW_H
