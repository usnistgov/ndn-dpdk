#ifndef NDN_DPDK_FETCH_SEG_H
#define NDN_DPDK_FETCH_SEG_H

/** @file */

#include "../mintmr/mintmr.h"

typedef TAILQ_ENTRY(FetchSeg) FetchRetxNode;

/** @brief Per-segment state. */
typedef struct FetchSeg
{
  uint64_t segNum;     ///< segment number
  TscTime txTime;      ///< last Interest tx time
  MinTmr rtoExpiry;    ///< RTO expiration timer
  FetchRetxNode retxQ; ///< retx queue node
  bool deleted_;       ///< (private for FetchWindow) whether seg has been deleted
  bool inRetxQ;        ///< whether segment is scheduled for retx
  uint16_t nRetx;      ///< number of Interest retx, increment upon TX
} __rte_cache_aligned FetchSeg;

static inline void
FetchSeg_Init(FetchSeg* seg, uint64_t segNum)
{
  seg->segNum = segNum;
  seg->txTime = 0;
  MinTmr_Init(&seg->rtoExpiry);
  seg->inRetxQ = false;
  seg->nRetx = 0;
}

#endif // NDN_DPDK_FETCH_SEG_H
