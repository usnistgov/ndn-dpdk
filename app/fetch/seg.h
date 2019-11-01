#ifndef NDN_DPDK_APP_FETCH_SEG_H
#define NDN_DPDK_APP_FETCH_SEG_H

/// \file

#include "../../container/mintmr/mintmr.h"

/** \brief Per-segment state.
 */
typedef struct FetchSeg FetchSeg;

struct FetchSeg
{
  TscTime txTime;     ///< last Interest tx time
  MinTmr rtoExpiry;   ///< RTO expiration timer
  FetchSeg* retxPrev; ///< retx queue prev
  FetchSeg* retxNext; ///< retx queue next
  bool deleted_; ///< (private for FetchWindow) whether seg has been deleted
  uint8_t nRetx; ///< number of Interest retx, excluding first Interest
} __rte_cache_aligned;

static inline void
FetchSeg_Init(FetchSeg* seg)
{
  seg->txTime = 0;
  MinTmr_Init(&seg->rtoExpiry);
  seg->retxPrev = seg->retxNext = NULL;
  seg->nRetx = 0;
}

#endif // NDN_DPDK_APP_FETCH_SEG_H
