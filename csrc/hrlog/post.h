#ifndef NDN_DPDK_HRLOG_POST_H
#define NDN_DPDK_HRLOG_POST_H

/** @file */

#include "entry.h"

#include <rte_lcore.h>
#include <rte_ring.h>

extern struct rte_ring* theHrlogRing;
static_assert(sizeof(HrlogEntry) == sizeof(void*), "");

static inline void
Hrlog_PostBulk(HrlogEntry* entries, uint16_t count)
{
  if (theHrlogRing != NULL) {
    rte_ring_enqueue_bulk(theHrlogRing, (void**)entries, count, NULL);
  }
}

#endif // NDN_DPDK_HRLOG_POST_H
