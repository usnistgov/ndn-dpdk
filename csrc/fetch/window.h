#ifndef NDN_DPDK_FETCH_WINDOW_H
#define NDN_DPDK_FETCH_WINDOW_H

/// \file

#include "seg.h"

/** \brief Window of segment states.
 */
typedef struct FetchWindow
{
  FetchSeg* array;
  uint32_t capacityMask; ///< array capacity minus one
  uint32_t loPos;        ///< array position for loSegNum
  uint64_t loSegNum;     ///< inclusive lower bound of segment numbers
  uint64_t hiSegNum;     ///< exclusive upper bound of segment numbers
} FetchWindow;

/** \brief Determine whether a segment number is in the window.
 */
static inline bool
FetchWindow_Contains_(FetchWindow* win, uint64_t segNum)
{
  return win->loSegNum <= segNum && segNum < win->hiSegNum;
}

/** \brief Access FetchSeg* of a segment number.
 *  \pre FetchWindow_Contains_(win, segNum)
 */
static inline FetchSeg*
FetchWindow_Access_(FetchWindow* win, uint64_t segNum)
{
  uint64_t pos = (segNum - win->loSegNum + win->loPos) & win->capacityMask;
  return &win->array[pos];
}

/** \brief Retrieve a segment's state.
 *  \retval NULL segment is not in the window or has been deleted.
 */
static inline FetchSeg*
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

/** \brief Create state for the next segment.
 *  \retval NULL window has reached its capacity limit.
 */
static inline FetchSeg*
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

/** \brief Move loPos and loSegNum after some segment states have been deleted.
 */
static __rte_noinline void
FetchWindow_Advance_(FetchWindow* win)
{
  while (win->loSegNum < win->hiSegNum) {
    FetchSeg* seg = &win->array[win->loPos];
    if (unlikely(!seg->deleted_)) {
      break;
    }
    win->loPos = (win->loPos + 1) & win->capacityMask;
    ++win->loSegNum;
  }
}

/** \brief Discard a segment's state.
 */
static inline void
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

#endif // NDN_DPDK_FETCH_WINDOW_H
