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

inline int
__SgFibNexthopFilter_Next(SgFibNexthopFilter filter, const SgFibEntry* entry,
                          int i, FaceId* nh)
{
  do {
    ++i;
  } while (i < (int)entry->nNexthops && (filter & (1 << i)));
  *nh = i < (int)entry->nNexthops ? entry->nexthops[i] : FACEID_INVALID;
  return i;
}

/** \brief Iterator over FIB nexthops that pass a filter.
 *  \param filter a SgFibNexthopFilter.
 *  \param entry pointer to SgFibEntry.
 *  \param index undeclared variable name for the entry.
 *  \param nh declared FaceId variable for nexthop face.
 *
 *  Example:
 *  \code
 *  FaceId nh;
 *  SgFibNexthopFilter_ForEach(filter, entry, i, nh) {
 *    // use i and nh
 *    // 'continue' and 'break' are available
 *  }
 *  \endcode
 */
#define SgFibNexthopFilter_ForEach(filter, entry, index, nh)                   \
  for (int index = __SgFibNexthopFilter_Next(filter, (entry), -1, &nh);        \
       index < (int)(entry)->nNexthops;                                        \
       index = __SgFibNexthopFilter_Next(filter, (entry), index, &nh))

#endif // NDN_DPDK_STRATEGY_API_FIB_H
