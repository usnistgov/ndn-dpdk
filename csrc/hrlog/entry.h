#ifndef NDNDPDK_HRLOG_ENTRY_H
#define NDNDPDK_HRLOG_ENTRY_H

/** @file */

#include "../core/common.h"
#include <rte_lcore.h>
#include <rte_ring.h>
#include <urcu-pointer.h>

/** @brief Action identifier in high resolution log. */
typedef enum HrlogAction {
  HRLOG_OI = 1, // Interest TX since RX
  HRLOG_OD = 2, // retrieved Data TX since RX
  HRLOG_OC = 4, // cached Data TX since Interest RX
} HrlogAction;

/** @brief A high resolution log entry. */
typedef uint64_t HrlogEntry;
static_assert(sizeof(HrlogEntry) == sizeof(void*), "");

static inline HrlogEntry
HrlogEntry_New(HrlogAction act, uint64_t value) {
  uint8_t lcore = rte_lcore_id();
  return (value << 16) | ((uint64_t)lcore << 8) | ((uint8_t)act);
}

/** @brief A high resolution log file header. */
typedef struct HrlogHeader {
  uint32_t magic;
  uint32_t version;
  uint64_t tschz;
} HrlogHeader;
static_assert(sizeof(HrlogHeader) == 16, "");

#define HRLOG_HEADER_MAGIC 0x35f0498a
#define HRLOG_HEADER_VERSION 2

/** @brief RCU-protected pointer to hrlog collector queue. */
typedef struct HrlogRingRef {
  struct rte_ring* r;
} HrlogRingRef;

extern HrlogRingRef theHrlogRing;

/**
 * @brief Return hrlog collector queue.
 * @pre Calling thread holds rcu_read_lock.
 * @retval NULL hrlog collection is disabled.
 */
static __rte_always_inline struct rte_ring*
HrlogRing_Get() {
  return rcu_dereference(theHrlogRing.r);
}

/** @brief Post entries to hrlog collector queue. */
__attribute__((nonnull)) static __rte_always_inline void
HrlogRing_Post(struct rte_ring* r, HrlogEntry* entries, uint16_t count) {
  if (count > 0) {
    rte_ring_enqueue_burst(r, (void**)entries, count, NULL);
  }
}

/**
 * @brief Post entries to hrlog collector queue if enabled.
 * @pre Calling thread holds rcu_read_lock.
 */
static __rte_always_inline void
Hrlog_Post(HrlogEntry* entries, uint16_t count) {
  if (count == 0) {
    return;
  }

  struct rte_ring* r = HrlogRing_Get();
  if (r != NULL) {
    HrlogRing_Post(r, entries, count);
  }
}

#endif // NDNDPDK_HRLOG_ENTRY_H
