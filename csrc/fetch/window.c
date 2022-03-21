#include "window.h"

void
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
