#ifndef NDN_DPDK_STRATEGYAPI_FIB_H
#define NDN_DPDK_STRATEGYAPI_FIB_H

/** @file */

#include "common.h"

#define SG_FIB_ENTRY_MAX_NEXTHOPS 8
#define SG_FIB_ENTRY_SCRATCH 96

typedef struct SgFibEntryDyn
{
  char a_[16];
} SgFibEntryDyn;

typedef struct SgFibEntry
{
  char a_[525];
  uint8_t nNexthops;
  char b_[2];
  FaceID nexthops[SG_FIB_ENTRY_MAX_NEXTHOPS];
  char c_[32];
  char scratch[SG_FIB_ENTRY_SCRATCH];
  char d_[96];
} SgFibEntry;

typedef uint32_t SgFibNexthopFilter;

/**
 * @brief Iterator of FIB nexthops passing a filter.
 *
 * @code
 * SgFibNexthopIt it;
 * for (SgFibNexthopIt_Init(&it, entry, filter); // or SgFibNexthopIt_Init2(&it, ctx)
 *      SgFibNexthopIt_Valid(&it);
 *      SgFibNexthopIt_Next(&it)) {
 *   int index = it.i;
 *   FaceID nexthop = it.nh;
 * }
 * @endcode
 */
typedef struct SgFibNexthopIt
{
  const SgFibEntry* entry;
  SgFibNexthopFilter filter;
  uint8_t i;
  FaceID nh;
} SgFibNexthopIt;

inline bool
SgFibNexthopIt_Valid(const SgFibNexthopIt* it)
{
  return it->i < it->entry->nNexthops;
}

inline void
SgFibNexthopIt_Advance_(SgFibNexthopIt* it)
{
  for (; SgFibNexthopIt_Valid(it); ++it->i) {
    if (it->filter & (1 << it->i)) {
      continue;
    }
    it->nh = it->entry->nexthops[it->i];
    return;
  }
  it->nh = 0;
}

inline void
SgFibNexthopIt_Init(SgFibNexthopIt* it, const SgFibEntry* entry, SgFibNexthopFilter filter)
{
  it->entry = entry;
  it->filter = filter;
  it->i = 0;
  SgFibNexthopIt_Advance_(it);
}

inline void
SgFibNexthopIt_Next(SgFibNexthopIt* it)
{
  ++it->i;
  SgFibNexthopIt_Advance_(it);
}

#endif // NDN_DPDK_STRATEGYAPI_FIB_H
