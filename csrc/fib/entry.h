#ifndef NDNDPDK_FIB_ENTRY_H
#define NDNDPDK_FIB_ENTRY_H

/** @file */

#include "../core/urcu.h"
#include "../iface/faceid.h"
#include "../strategycode/strategy-code.h"
#include <urcu/rculfhash.h>

enum
{
  FibMaxNameLength = 494,
  FibMaxNexthops = 8,
  FibScratchSize = 96,
};

typedef struct FibEntry FibEntry;

/** @brief A FIB entry. */
struct FibEntry
{
  struct cds_lfht_node lfhtnode;
  char copyBegin_[0];
  uint16_t nameL; ///< TLV-LENGTH of name
  uint8_t nameV[FibMaxNameLength];
  char cachelineA_[0];

  union
  {
    /**
     * @brief Forwarding strategy.
     * @pre maxDepth == 0
     */
    StrategyCode* strategy;

    /**
     * @brief Real FIB entry.
     * @pre maxDepth > 0
     */
    FibEntry* realEntry;
  };

  uint32_t seqNum; ///< sequence number to detect FIB changes

  uint8_t nComps;    ///< number of name components
  uint8_t nNexthops; ///< number of nexthops

  /**
   * @brief Maximum potential LPM match relative to this entry.
   *
   * This field is known as '(MD - M)' in 2-stage LPM algorithm.
   * This number must be no less than the depth of all FIB entries whose name starts
   * with the name of this FIB entry, minus the depth of this entry.
   * 'depth' means number of name components.
   *
   * @pre nComps == startDepth
   */
  uint8_t maxDepth;

  FaceID nexthops[FibMaxNexthops];

  uint32_t nRxInterests;
  uint32_t nRxData;
  uint32_t nRxNacks;
  uint32_t nTxInterests;
  char copyEnd_[0];
  char padB_[16];
  char cachelineB_[0];

  char scratch[FibScratchSize];
  char padC_[32];
  char cachelineC_[0];
  struct rcu_head rcuhead;
} __rte_cache_aligned;

// FibEntry.nComps must be able to represent maximum number of name components that
// can fit in FibMaxNameLength octets.
static_assert(UINT8_MAX >= FibMaxNameLength / 2, "");

static_assert(offsetof(FibEntry, cachelineA_) % RTE_CACHE_LINE_SIZE == 0, "");
static_assert(offsetof(FibEntry, cachelineB_) % RTE_CACHE_LINE_SIZE == 0, "");
static_assert(offsetof(FibEntry, cachelineC_) % RTE_CACHE_LINE_SIZE == 0, "");

__attribute__((nonnull)) static inline void
FibEntry_Copy(FibEntry* dst, const FibEntry* src)
{
  rte_memcpy(dst->copyBegin_, src->copyBegin_,
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

static inline FibEntry**
FibEntry_PtrRealEntry(FibEntry* entry)
{
  return &entry->realEntry;
}

static inline StrategyCode**
FibEntry_PtrStrategy(FibEntry* entry)
{
  return &entry->strategy;
}

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
FibNexthopFilter_Reject(FibNexthopFilter* filter, const FibEntry* entry, FaceID nh)
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

#endif // NDNDPDK_FIB_ENTRY_H
