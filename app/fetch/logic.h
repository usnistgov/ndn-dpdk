#ifndef NDN_DPDK_APP_FETCH_LOGIC_H
#define NDN_DPDK_APP_FETCH_LOGIC_H

/// \file

#include "rttest.h"
#include "tcpcubic.h"
#include "window.h"

typedef TAILQ_HEAD(FetchRetxQueue, FetchSeg) FetchRetxQueue;

/** \brief Fetcher congestion control and scheduling logic.
 */
typedef struct FetchLogic
{
  FetchWindow win;
  RttEst rtte;
  TcpCubic ca;
  FetchRetxQueue retxQ;
  MinSched* sched;
  uint64_t finalSegNum;
  uint64_t lastCwndDecreaseSegNum;
  uint32_t nInFlight;
} FetchLogic;

void
FetchLogic_Init_(FetchLogic* fl);

/** \brief Set final segment number.
 *  \param segNum segment number of the last segment, inclusive.
 */
static inline void
FetchLogic_SetFinalSegNum(FetchLogic* fl, uint64_t segNum)
{
  fl->finalSegNum = segNum;
}

/** \brief Determine if all segments have been fetched.
 */
static inline bool
FetchLogic_Finished(FetchLogic* fl)
{
  return fl->win.loSegNum > fl->finalSegNum;
}

/** \brief Request to transmit a burst of Interests.
 *  \param[out] segNums segment numbers to retrieve.
 *  \param limit size of segNums array.
 */
size_t
FetchLogic_TxInterestBurst(FetchLogic* fl, uint64_t* segNums, size_t limit);

/** \brief Notify Data arrival.
 *  \param segNums segment numbers in arrived Data.
 *  \param count size of segNums array.
 */
void
FetchLogic_RxDataBurst(FetchLogic* fl, const uint64_t* segNums, size_t count);

#endif // NDN_DPDK_APP_FETCH_LOGIC_H
