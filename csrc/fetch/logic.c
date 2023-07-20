#include "logic.h"

#include "../core/logger.h"

N_LOG_INIT(FetchLogic);

__attribute__((nonnull)) static __rte_always_inline FetchSeg*
FetchLogic_TxInterest(FetchLogic* fl, int* nNew, int* nRetx) {
  if (!cds_list_empty(&fl->retxQ)) { // drain retxQ
    FetchSeg* seg = cds_list_first_entry(&fl->retxQ, FetchSeg, retxNode);
    cds_list_del(&seg->retxNode);

    if (likely(seg->segNum < fl->segmentEnd)) {
      seg->inRetxQ = false;
      seg->hasRetx = true;
      ++*nRetx;
      MinTmr_Cancel(&seg->rtoExpiry);
      return seg;
    }

    FetchWindow_Delete(&fl->win, seg->segNum);
  }

  if (unlikely(fl->win.hiSegNum >= fl->segmentEnd)) { // reached final segment
    return NULL;
  }

  FetchSeg* seg = FetchWindow_Append(&fl->win); // fetch new segment
  if (likely(seg != NULL)) {
    ++*nNew;
    // seg->rtoExpiry is zero'ed, safe to pass to MinTmr_After
    return seg;
  }

  // FetchWindow is full
  return NULL;
}

size_t
FetchLogic_TxInterestBurst(FetchLogic* fl, uint64_t* segNums, size_t limit, TscTime now) {
  uint32_t cwnd = TcpCubic_GetCwnd(&fl->ca);
  size_t count = 0;
  int nNew = 0, nRetx = 0;

  while (fl->nInFlight < cwnd && count < limit) {
    FetchSeg* seg = FetchLogic_TxInterest(fl, &nNew, &nRetx);
    if (unlikely(seg == NULL)) {
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

__attribute__((nonnull)) static inline bool
FetchLogic_DecreaseCwnd(FetchLogic* fl, const char* caller, uint64_t segNum, TscTime now) {
  if (unlikely(fl->hiDataSegNum <= fl->cwndDecInterestSegNum)) {
    return false;
  }
  TcpCubic_Decrease(&fl->ca, now);
  fl->cwndDecInterestSegNum = fl->win.hiSegNum;

  N_LOGD("%s fl=%p seg=%" PRIu64 " win=[%" PRIu64 ",%" PRIu64 ") hi-data=%" PRIu64 " rto=%" PRId64
         " cwnd=%" PRIu32 " nInFlight=%" PRIu32 "",
         caller, fl, segNum, fl->win.loSegNum, fl->win.hiSegNum, fl->hiDataSegNum,
         TscDuration_ToMillis(fl->rtte.rto), TcpCubic_GetCwnd(&fl->ca), fl->nInFlight);
  return true;
}

__attribute__((nonnull)) static inline void
FetchLogic_RxData(FetchLogic* fl, TscTime now, uint64_t segNum, bool hasCongMark,
                  bool isFinalBlock) {
  FetchSeg* seg = FetchWindow_Get(&fl->win, segNum);
  if (unlikely(seg == NULL)) {
    return;
  }
  ++fl->nRxData;

  if (unlikely(seg->inRetxQ)) {
    // cancel retransmission
    cds_list_del_init(&seg->retxNode);
  } else {
    // cancel RTO timer
    --fl->nInFlight;
    MinTmr_Cancel(&seg->rtoExpiry);
  }

  if (likely(!seg->hasRetx)) { // RTT valid only if no retx was sent
    TscDuration rtt = ((uint64_t)now - seg->txTime) & FetchSegTxTimeMask;
    RttEst_Push(&fl->rtte, now, rtt);
  }

  if (unlikely(hasCongMark)) {
    FetchLogic_DecreaseCwnd(fl, "RxDataCongMark", segNum, now);
  } else {
    TcpCubic_Increase(&fl->ca, now, fl->rtte.rttv.sRtt);
  }

  if (unlikely(isFinalBlock)) {
    fl->segmentEnd = segNum + 1;
  }

  fl->hiDataSegNum = RTE_MAX(fl->hiDataSegNum, segNum);
  FetchWindow_Delete(&fl->win, segNum);
}

void
FetchLogic_RxDataBurst(FetchLogic* fl, const FetchLogicRxData* pkts, size_t count, TscTime now) {
  for (size_t i = 0; i < count; ++i) {
    FetchLogic_RxData(fl, now, pkts[i].segNum, pkts[i].congMark > 0, pkts[i].isFinalBlock);
  }
  if (unlikely(fl->finishTime == 0 && fl->win.loSegNum >= fl->segmentEnd)) {
    fl->finishTime = rte_get_tsc_cycles();
  }
}

__attribute__((nonnull)) static void
FetchLogic_RtoTimeout(MinTmr* tmr, uintptr_t flPtr) {
  FetchLogic* fl = (FetchLogic*)flPtr;
  FetchSeg* seg = container_of(tmr, FetchSeg, rtoExpiry);

  --fl->nInFlight;

  if (unlikely(seg->segNum >= fl->segmentEnd)) {
    FetchWindow_Delete(&fl->win, seg->segNum);
    return;
  }

  if (FetchLogic_DecreaseCwnd(fl, "RtoTimeout", seg->segNum, rte_get_tsc_cycles())) {
    RttEst_Backoff(&fl->rtte);
  }

  seg->inRetxQ = true;
  cds_list_add_tail(&seg->retxNode, &fl->retxQ);
}

void
FetchLogic_Init(FetchLogic* fl, uint32_t winCapacity, int numaSocket) {
  FetchWindow_Init(&fl->win, winCapacity, numaSocket);

  // 2^16 slots of 1ms interval, accommodates RTO up to 65536ms
  fl->sched = MinSched_New(16, TscHz / 1000, FetchLogic_RtoTimeout, (uintptr_t)fl);
  NDNDPDK_ASSERT(MinSched_GetMaxDelay(fl->sched) >= RttEstTscMaxRto);

  FetchLogic_Reset(fl, 0, UINT64_MAX);
}

void
FetchLogic_Free(FetchLogic* fl) {
  MinSched_Close(fl->sched);
  FetchWindow_Free(&fl->win);
}

void
FetchLogic_Reset(FetchLogic* fl, uint64_t segmentBegin, uint64_t segmentEnd) {
  FetchWindow_Reset(&fl->win, segmentBegin);
  RttEst_Init(&fl->rtte);
  TcpCubic_Init(&fl->ca);
  MinSched_Clear(fl->sched);

  CDS_INIT_LIST_HEAD(&fl->retxQ);
  fl->segmentEnd = segmentEnd;
  fl->startTime = rte_get_tsc_cycles();
  fl->finishTime = 0;
  fl->nTxRetx = 0;
  fl->nRxData = 0;
  fl->hiDataSegNum = 0;
  fl->cwndDecInterestSegNum = 0;
  fl->nInFlight = 0;
}
