#ifndef NDNDPDK_STRATEGYAPI_FIB_H
#define NDNDPDK_STRATEGYAPI_FIB_H

/** @file */

#include "../core/rttest.h"
#include "../fib/enum.h"
#include "common.h"

typedef struct SgFibEntryDyn {
  uint8_t a_[32];
  uint8_t scratch[FibScratchSize];
  RttValue rtt[FibMaxNexthops];
} SgFibEntryDyn;

typedef struct SgFibEntry {
  uint8_t a_[525];
  uint8_t nNexthops;
  uint8_t b_[2];
  FaceID nexthops[FibMaxNexthops];
} SgFibEntry;

typedef uint32_t SgFibNexthopFilter;

SUBROUTINE bool
SgFibNexthopFilter_Rejected(SgFibNexthopFilter filter, uint8_t i) {
  static_assert(__builtin_types_compatible_p(typeof(filter), uint32_t), "");
  return rte_bit_test(&filter, i);
}

/**
 * @brief Iterator of FIB nexthops passing a filter.
 *
 * @code
 * SgFibNexthopIt it;
 * for (SgFibNexthopIt_Init(&it, entry, filter); // or SgFibNexthopIt_InitCtx(&it, ctx)
 *      SgFibNexthopIt_Valid(&it);
 *      SgFibNexthopIt_Next(&it)) {
 *   int index = it.i;
 *   FaceID nexthop = it.nh;
 * }
 * @endcode
 */
typedef struct SgFibNexthopIt {
  const SgFibEntry* entry;
  SgFibNexthopFilter filter;
  uint8_t i;
  FaceID nh;
} SgFibNexthopIt;

SUBROUTINE bool
SgFibNexthopIt_Valid(const SgFibNexthopIt* it) {
  return it->i < it->entry->nNexthops;
}

SUBROUTINE void
SgFibNexthopIt_Advance_(SgFibNexthopIt* it) {
  for (; SgFibNexthopIt_Valid(it); ++it->i) {
    if (SgFibNexthopFilter_Rejected(it->filter, it->i)) {
      continue;
    }
    it->nh = it->entry->nexthops[it->i];
    return;
  }
  it->nh = 0;
}

SUBROUTINE void
SgFibNexthopIt_Init(SgFibNexthopIt* it, const SgFibEntry* entry, SgFibNexthopFilter filter) {
  it->entry = entry;
  it->filter = filter;
  it->i = 0;
  SgFibNexthopIt_Advance_(it);
}

SUBROUTINE void
SgFibNexthopIt_Next(SgFibNexthopIt* it) {
  ++it->i;
  SgFibNexthopIt_Advance_(it);
}

#endif // NDNDPDK_STRATEGYAPI_FIB_H
