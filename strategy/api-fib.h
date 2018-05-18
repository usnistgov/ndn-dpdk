#ifndef NDN_DPDK_STRATEGY_API_FIB_H
#define NDN_DPDK_STRATEGY_API_FIB_H

/// \file

#include "api-common.h"

#define SG_FIB_ENTRY_MAX_NEXTHOPS 8
#define SG_FIB_DYN_SCRATCH 96

typedef struct SgFibEntryDyn
{
  char _a[16];
  char scratch[SG_FIB_DYN_SCRATCH];
} SgFibEntryDyn;

typedef struct SgFibEntry
{
  char _a[8];
  SgFibEntryDyn* dyn;
  char _b[3];
  uint8_t nNexthops;
  char _c[2];
  FaceId nexthops[SG_FIB_ENTRY_MAX_NEXTHOPS];
  char _d[500];
} SgFibEntry;

typedef uint32_t SgFibNexthopFilter;

/** \brief Iterator of FIB nexthops passing a filter.
 *
 *  \code
 *  SgFibNexthopIt it;
 *  for (SgFibNexthopIt_Init(&it, entry, filter); // or SgFibNexthopIt_Init2(&it, ctx)
 *       SgFibNexthopIt_Valid(&it);
 *       SgFibNexthopIt_Next(&it)) {
 *    int index = it.i;
 *    FaceId nexthop = it.nh;
 *  }
 *  \endcode
 */
typedef struct SgFibNexthopIt
{
  const SgFibEntry* entry;
  SgFibNexthopFilter filter;
  uint8_t i;
  FaceId nh;
} SgFibNexthopIt;

inline bool
SgFibNexthopIt_Valid(const SgFibNexthopIt* it)
{
  return it->i < it->entry->nNexthops;
}

inline void
__SgFibNexthopIt_Advance(SgFibNexthopIt* it)
{
  for (; SgFibNexthopIt_Valid(it); ++it->i) {
    if (it->filter & (1 << it->i)) {
      continue;
    }
    it->nh = it->entry->nexthops[it->i];
    return;
  }
  it->nh = FACEID_INVALID;
}

inline void
SgFibNexthopIt_Init(SgFibNexthopIt* it, const SgFibEntry* entry,
                    SgFibNexthopFilter filter)
{
  it->entry = entry;
  it->filter = filter;
  it->i = 0;
  __SgFibNexthopIt_Advance(it);
}

inline void
SgFibNexthopIt_Next(SgFibNexthopIt* it)
{
  ++it->i;
  __SgFibNexthopIt_Advance(it);
}

#endif // NDN_DPDK_STRATEGY_API_FIB_H
