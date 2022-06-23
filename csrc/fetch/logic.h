#ifndef NDNDPDK_FETCH_LOGIC_H
#define NDNDPDK_FETCH_LOGIC_H

/** @file */

#include "../core/rttest.h"
#include "tcpcubic.h"
#include "window.h"

typedef TAILQ_HEAD(FetchRetxQueue, FetchSeg) FetchRetxQueue;

/** @brief Fetcher congestion control and scheduling logic. */
typedef struct FetchLogic
{
  FetchWindow win;
  RttEst rtte;
  TcpCubic ca;
  struct cds_list_head retxQ;
  MinSched* sched;
  TscTime startTime;
  TscTime finishTime;
  uint64_t nTxRetx;
  uint64_t nRxData;
  uint64_t finalSegNum;
  uint64_t hiDataSegNum;
  uint64_t cwndDecreaseInterestSegNum;
  uint32_t nInFlight;
} FetchLogic;

__attribute__((nonnull)) void
FetchLogic_Init_(FetchLogic* fl);

/**
 * @brief Request to transmit a burst of Interests.
 * @param[out] segNums segment numbers to retrieve.
 * @param limit size of segNums array.
 */
__attribute__((nonnull)) size_t
FetchLogic_TxInterestBurst(FetchLogic* fl, uint64_t* segNums, size_t limit, TscTime now);

typedef struct FetchLogicRxData
{
  uint64_t segNum;
  uint8_t congMark;
} FetchLogicRxData;

/**
 * @brief Notify Data arrival.
 * @param pkts fields extracted from arrived Data.
 * @param count size of @p pkts array.
 */
__attribute__((nonnull)) void
FetchLogic_RxDataBurst(FetchLogic* fl, const FetchLogicRxData* pkts, size_t count, TscTime now);

#endif // NDNDPDK_FETCH_LOGIC_H
