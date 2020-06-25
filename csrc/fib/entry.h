#ifndef NDN_DPDK_FIB_ENTRY_H
#define NDN_DPDK_FIB_ENTRY_H

/// \file

#include "entry-struct.h"

static inline void
FibEntry_Copy(FibEntry* dst, const FibEntry* src)
{
  rte_memcpy(dst->copyBegin_,
             src->copyBegin_,
             offsetof(FibEntry, copyEnd_) - offsetof(FibEntry, copyBegin_));
}

static inline FibEntry*
FibEntry_GetReal(FibEntry* entry)
{
  if (unlikely(entry == NULL) || likely(entry->maxDepth == 0)) {
    return entry;
  }
  return entry->realEntry;
}

/** \brief A filter over FIB nexthops.
 *
 *  The zero value permits all nexthops in the FIB entry.
 */
typedef uint32_t FibNexthopFilter;

static_assert(CHAR_BIT * sizeof(FibNexthopFilter) >= FIB_ENTRY_MAX_NEXTHOPS,
              "");

/** \brief Reject the given nexthop.
 *  \param[inout] filter original and updated filter.
 *  \return how many nexthops pass the filter after the update.
 */
static inline int
FibNexthopFilter_Reject(FibNexthopFilter* filter,
                        const FibEntry* entry,
                        FaceId nh)
{
  int count = 0;
  for (uint8_t i = 0; i < entry->nNexthops; ++i) {
    if (entry->nexthops[i] == nh) {
      *filter |= (1 << i);
    }
    if (!(*filter & (1 << i))) {
      ++count;
    }
  }
  return count;
}

#endif // NDN_DPDK_FIB_ENTRY_H
