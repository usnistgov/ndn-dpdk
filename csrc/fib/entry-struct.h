#ifndef NDN_DPDK_FIB_ENTRY_STRUCT_H
#define NDN_DPDK_FIB_ENTRY_STRUCT_H

/** @file */

#include "../core/urcu.h"
#include "../iface/faceid.h"
#include "../strategycode/strategy-code.h"
#include <urcu/rculfhash.h>

#define FIB_ENTRY_MAX_NAME_LEN 494
#define FIB_ENTRY_MAX_NEXTHOPS 8
#define FIB_ENTRY_SCRATCH 96

typedef struct FibEntry FibEntry;

/** @brief A FIB entry. */
struct FibEntry
{
  struct cds_lfht_node lfhtnode;
  char copyBegin_[0];
  uint16_t nameL; ///< TLV-LENGTH of name
  uint8_t nameV[FIB_ENTRY_MAX_NAME_LEN];
  char cachelineA_[0];

#ifdef GODEF
  void* union_strategy_realEntry;
#else
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
#endif

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

  FaceID nexthops[FIB_ENTRY_MAX_NEXTHOPS];

  uint32_t nRxInterests;
  uint32_t nRxData;
  uint32_t nRxNacks;
  uint32_t nTxInterests;
  char copyEnd_[0];
  char padB_[16];
  char cachelineB_[0];

  char scratch[FIB_ENTRY_SCRATCH];
  char padC_[32];
  char cachelineC_[0];
  struct rcu_head rcuhead;
} __rte_cache_aligned;

// FibEntry.nComps must be able to represent maximum number of name components that
// can fit in FIB_ENTRY_MAX_NAME_LEN octets.
static_assert(UINT8_MAX >= FIB_ENTRY_MAX_NAME_LEN / 2, "");

static_assert(offsetof(FibEntry, cachelineA_) % RTE_CACHE_LINE_SIZE == 0, "");
static_assert(offsetof(FibEntry, cachelineB_) % RTE_CACHE_LINE_SIZE == 0, "");
static_assert(offsetof(FibEntry, cachelineC_) % RTE_CACHE_LINE_SIZE == 0, "");

#endif // NDN_DPDK_FIB_ENTRY_STRUCT_H
