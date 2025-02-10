#ifndef NDNDPDK_FIB_NEXTHOP_FILTER_H
#define NDNDPDK_FIB_NEXTHOP_FILTER_H

/** @file */

#include "entry.h"

/**
 * @brief A filter over FIB nexthops.
 *
 * The zero value permits all nexthops in the FIB entry.
 */
typedef uint32_t FibNexthopFilter;

static_assert(CHAR_BIT * sizeof(FibNexthopFilter) >= FibMaxNexthops, "");

/**
 * @brief Reject the given nexthop.
 * @param[inout] filter original and updated filter.
 * @return how many nexthops pass the filter after the update.
 */
__attribute__((nonnull)) static inline int
FibNexthopFilter_Reject(FibNexthopFilter* filter, const FibEntry* entry, FaceID nh) {
  for (uint8_t i = 0; i < entry->nNexthops; ++i) {
    if (entry->nexthops[i] == nh) {
      rte_bit_set(filter, i);
      break;
    }
  }
  static_assert(__builtin_types_compatible_p(typeof(*filter), uint32_t), "");
  return entry->nNexthops - rte_popcount32(*filter);
}

#endif // NDNDPDK_FIB_NEXTHOP_FILTER_H
