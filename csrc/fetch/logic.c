#include "logic.h"

#include "../core/logger.h"

N_LOG_INIT(FetchLogic);

size_t
FetchLogic_TxInterestBurst(FetchLogic* fl, uint64_t* segNums, size_t limit)
{
  TscTime now = rte_get_tsc_cycles();
  uint32_t cwnd = TcpCubic_GetCwnd(&fl->ca);
  size_t count = 0;
  int nNew = 0, nRetx = 0;

  while (fl->nInFlight < cwnd && count < limit) {
    FetchSeg* seg;
    if ((seg = TAILQ_FIRST(&fl->retxQ)) != NULL) { // drain retxQ first
      TAILQ_REMOVE(&fl->retxQ, seg, retxQ);
      seg->inRetxQ = false;
      ++seg->nRetx;
      ++nRetx;
    } else if (unlikely(fl->win.hiSegNum > fl->finalSegNum)) { // reached final segment
      break;
    } else if (likely((seg = FetchWindow_Append(&fl->win)) != NULL)) { // fetch new segment
      ++nNew;
    } else { // FetchWindow is full
      break;
    }

    seg->txTime = now;
    bool ok = MinTmr_After(&seg->rtoExpiry, fl->rtte.rto, fl->sched);
    NDNDPDK_ASSERT(ok);
    segNums[count++] = seg->segNum;
    ++fl->nInFlight;
  }

  if (likely(count > 0)) {
    N_LOGV("TX fl=%p new=%d retx=%d win=[%" PRIu64 ",%" PRIu64 ") rto=%" PRId64 " cwnd=%" PRIu32
           " nInFlight=%" PRIu32 "",
           fl, nNew, nRetx, fl->win.loSegNum, fl->win.hiSegNum, TscDuration_ToMillis(fl->rtte.rto),
           cwnd, fl->nInFlight);
  }
  fl->nTxRetx += nRetx;

  return count;
}

static inline bool
FetchLogic_DecreaseCwnd(FetchLogic* fl, const char* caller, uint64_t segNum, TscTime now)
{
  if (unlikely(fl->hiDataSegNum <= fl->cwndDecreaseInterestSegNum)) {
    return false;
  }
  TcpCubic_Decrease(&fl->ca, now);
  fl->cwndDecreaseInterestSegNum = fl->win.hiSegNum;

  N_LOGD("%s fl=%p seg=%" PRIu64 " win=[%" PRIu64 ",%" PRIu64 ") hi-data=%" PRIu64 " rto=%" PRId64
         " cwnd=%" PRIu32 " nInFlight=%" PRIu32 "",
         caller, fl, segNum, fl->win.loSegNum, fl->win.hiSegNum, fl->hiDataSegNum,
         TscDuration_ToMillis(fl->rtte.rto), TcpCubic_GetCwnd(&fl->ca), fl->nInFlight);
  return true;
}

static inline void
FetchLogic_RxData(FetchLogic* fl, TscTime now, uint64_t segNum, bool hasCongMark)
{
  FetchSeg* seg = FetchWindow_Get(&fl->win, segNum);
  if (unlikely(seg == NULL)) {
    return;
  }
  ++fl->nRxData;

  if (unlikely(seg->inRetxQ)) {
    // cancel retransmission
    TAILQ_REMOVE(&fl->retxQ, seg, retxQ);
  } else {
    // cancel RTO timer
    --fl->nInFlight;
    MinTmr_Cancel(&seg->rtoExpiry);
  }

  if (likely(seg->nRetx == 0)) {
    // RTT valid only if no retx was sent
    TscDuration rtt = now - seg->txTime;
    RttEst_Push(&fl->rtte, now, rtt);
  }

  if (unlikely(hasCongMark)) {
    FetchLogic_DecreaseCwnd(fl, "RxDataCongMark", segNum, now);
  } else {
    TcpCubic_Increase(&fl->ca, now, fl->rtte.sRtt);
  }

  fl->hiDataSegNum = RTE_MAX(fl->hiDataSegNum, segNum);
  FetchWindow_Delete(&fl->win, segNum);
}

void
FetchLogic_RxDataBurst(FetchLogic* fl, const FetchLogicRxData* pkts, size_t count)
{
  TscTime now = rte_get_tsc_cycles();
  for (size_t i = 0; i < count; ++i) {
    FetchLogic_RxData(fl, now, pkts[i].segNum, pkts[i].congMark > 0);
  }
}

static void
FetchLogic_RtoTimeout(MinTmr* tmr, void* cbarg)
{
  FetchLogic* fl = (FetchLogic*)cbarg;
  FetchSeg* seg = container_of(tmr, FetchSeg, rtoExpiry);

  --fl->nInFlight;

  if (unlikely(seg->segNum > fl->finalSegNum)) {
    return;
  }

  if (FetchLogic_DecreaseCwnd(fl, "RtoTimeout", seg->segNum, rte_get_tsc_cycles())) {
    RttEst_Backoff(&fl->rtte);
  }

  seg->inRetxQ = true;
  TAILQ_INSERT_TAIL(&fl->retxQ, seg, retxQ);
}

void
FetchLogic_Init_(FetchLogic* fl)
{
  NDNDPDK_ASSERT(rte_align32pow2(fl->win.capacityMask) - 1 == fl->win.capacityMask);

  TAILQ_INIT(&fl->retxQ);
  fl->nTxRetx = 0;
  fl->nRxData = 0;
  fl->finalSegNum = UINT64_MAX;
  fl->hiDataSegNum = 0;
  fl->cwndDecreaseInterestSegNum = 0;
  fl->nInFlight = 0;

  // 2^16 slots of 1ms interval, accommodates RTO up to 65536ms
  fl->sched = MinSched_New(16, TscHz / 1000, FetchLogic_RtoTimeout, fl);
  NDNDPDK_ASSERT(MinSched_GetMaxDelay(fl->sched) >= (TscDuration)(RTTEST_MAXRTO_MS * TscHz / 1000));
}
