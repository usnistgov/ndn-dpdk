#ifndef NDN_DPDK_CONTAINER_FIB_ENTRY_H
#define NDN_DPDK_CONTAINER_FIB_ENTRY_H

/// \file

#include "../../iface/faceid.h"
#include "../strategycode/strategy-code.h"

#define FIB_ENTRY_MAX_NAME_LEN 500
#define FIB_ENTRY_MAX_NEXTHOPS 8
#define FIB_DYN_SCRATCH 96

/** \brief Counters and strategy scratch area on FIB entry.
 */
typedef struct FibEntryDyn
{
  uint32_t nRxInterests;
  uint32_t nRxData;
  uint32_t nRxNacks;
  uint32_t nTxInterests;

  char scratch[FIB_DYN_SCRATCH];
} FibEntryDyn;

static void
FibEntryDyn_Copy(FibEntryDyn* dst, const FibEntryDyn* src)
{
  rte_memcpy(dst, src, offsetof(FibEntryDyn, scratch));
  memset(dst->scratch, 0, sizeof(dst->scratch));
}

/** \brief A FIB entry.
 */
typedef struct FibEntry
{
  StrategyCode* strategy;
  FibEntryDyn* dyn;

  uint32_t seqNum; ///< sequence number to detect FIB changes

  uint16_t nameL;    ///< TLV-LENGTH of name
  uint8_t nComps;    ///< number of name components
  uint8_t nNexthops; ///< number of nexthops

  /** \brief maximum potential LPM match relative to this entry
   *
   *  This field is known as '(MD - M)' in 2-stage LPM paper.
   *  This number must be no less than the depth of all FIB entries whose name starts
   *  with the name of this FIB entry, minus the depth of this entry.
   *  'depth' means number of name components.
   */
  uint8_t maxDepth;

  bool shouldFreeDyn; ///< (private) read by Fib_FinalizeEntry

  FaceId nexthops[FIB_ENTRY_MAX_NEXTHOPS];
  uint8_t nameV[FIB_ENTRY_MAX_NAME_LEN];
} FibEntry;

// FibEntry.nComps must be able to represent maximum number of name components that
// can fit in FIB_ENTRY_MAX_NAME_LEN octets.
static_assert(UINT8_MAX >= FIB_ENTRY_MAX_NAME_LEN / 2, "");

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
static int
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

#endif // NDN_DPDK_CONTAINER_FIB_ENTRY_H
