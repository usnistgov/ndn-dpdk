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
  bool deleted_;                 ///< (private for FetchWindow) whether seg has been deleted
} __rte_cache_aligned FetchSeg;

__attribute__((nonnull)) static inline void
FetchSeg_Init(FetchSeg* seg, uint64_t segNum)
{
  seg->segNum = segNum;
  seg->txTime = 0;
  MinTmr_Init(&seg->rtoExpiry);
  CDS_INIT_LIST_HEAD(&seg->retxNode);
  seg->nRetx = 0;
  seg->deleted_ = false;
}

/** @brief Window of segment states. */
typedef struct FetchWindow
{
  FetchSeg* array;
  uint32_t capacityMask; ///< array capacity minus one
  uint32_t loPos;        ///< array position for loSegNum
  uint64_t loSegNum;     ///< inclusive lower bound of segment numbers
  uint64_t hiSegNum;     ///< exclusive upper bound of segment numbers
} FetchWindow;

/** @brief Determine whether a segment number is in the window. */
__attribute__((nonnull)) static inline bool
FetchWindow_Contains_(FetchWindow* win, uint64_t segNum)
{
  return win->loSegNum <= segNum && segNum < win->hiSegNum;
}

/**
 * @brief Access FetchSeg* of a segment number.
 * @pre FetchWindow_Contains_(win, segNum)
 */
__attribute__((nonnull, returns_nonnull)) static inline FetchSeg*
FetchWindow_Access_(FetchWindow* win, uint64_t segNum)
{
  uint64_t pos = (segNum - win->loSegNum + win->loPos) & win->capacityMask;
  return &win->array[pos];
}

/**
 * @brief Retrieve a segment's state.
 * @retval NULL segment is not in the window or has been deleted.
 */
__attribute__((nonnull)) static inline FetchSeg*
FetchWindow_Get(FetchWindow* win, uint64_t segNum)
{
  if (unlikely(!FetchWindow_Contains_(win, segNum))) {
    return NULL;
  }
  FetchSeg* seg = FetchWindow_Access_(win, segNum);
  if (unlikely(seg->deleted_)) {
    return NULL;
  }
  return seg;
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
  FetchSeg* seg = FetchWindow_Access_(win, segNum);
  seg->deleted_ = false;
  FetchSeg_Init(seg, segNum);
  return seg;
}

/** @brief Move loPos and loSegNum after some segment states have been deleted. */
__attribute__((nonnull)) void
FetchWindow_Advance_(FetchWindow* win);

/** @brief Discard a segment's state. */
__attribute__((nonnull)) static inline void
FetchWindow_Delete(FetchWindow* win, uint64_t segNum)
{
  FetchSeg* seg = FetchWindow_Get(win, segNum);
  if (unlikely(seg == NULL)) {
    return;
  }
  seg->deleted_ = true;
  if (segNum == win->loSegNum) {
    FetchWindow_Advance_(win);
  }
}

#endif // NDNDPDK_FETCH_WINDOW_H
