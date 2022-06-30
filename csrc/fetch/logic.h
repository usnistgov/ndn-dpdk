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
  uint64_t segmentEnd;            ///< last segnum desired plus one
  TscTime startTime;              ///< start time
  TscTime finishTime;             ///< finish time, 0 if not finished
  uint64_t nTxRetx;               ///< retransmitted Interests
  uint64_t nRxData;               ///< non-duplicate Data
  uint64_t hiDataSegNum;          ///< highest Data segnum received
  uint64_t cwndDecInterestSegNum; ///< highest Interest segnum when cwnd was last decreased
  uint32_t nInFlight;             ///< count of in-flight Interests
} FetchLogic;

__attribute__((nonnull)) void
FetchLogic_Init(FetchLogic* fl, uint32_t winCapacity, int numaSocket);

__attribute__((nonnull)) void
FetchLogic_Free(FetchLogic* fl);

__attribute__((nonnull)) void
FetchLogic_Reset(FetchLogic* fl, uint64_t segmentBegin, uint64_t segmentEnd);

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
  bool isFinalBlock;
} FetchLogicRxData;

/**
 * @brief Notify Data arrival.
 * @param pkts fields extracted from arrived Data.
 * @param count size of @p pkts array.
 */
__attribute__((nonnull)) void
FetchLogic_RxDataBurst(FetchLogic* fl, const FetchLogicRxData* pkts, size_t count, TscTime now);

#endif // NDNDPDK_FETCH_LOGIC_H
