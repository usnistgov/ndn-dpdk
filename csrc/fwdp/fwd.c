#include "fwd.h"
#include "strategy.h"

#include "../core/logger.h"

INIT_ZF_LOG(FwFwd);

static_assert((int)SGEVT_INTEREST == (int)PktInterest, "");
static_assert((int)SGEVT_DATA == (int)PktData, "");
static_assert((int)SGEVT_NACK == (int)PktNack, "");
static_assert(offsetof(SgCtx, global) == offsetof(FwFwdCtx, fwd), "");
static_assert(offsetof(FwFwd, sgGlobal) == 0, "");
static_assert(offsetof(SgCtx, now) == offsetof(FwFwdCtx, rxTime), "");
static_assert(offsetof(SgCtx, eventKind) == offsetof(FwFwdCtx, eventKind), "");
static_assert(offsetof(SgCtx, nhFlt) == offsetof(FwFwdCtx, nhFlt), "");
static_assert(offsetof(SgCtx, pkt) == offsetof(FwFwdCtx, pkt), "");
static_assert(offsetof(SgCtx, fibEntry) == offsetof(FwFwdCtx, fibEntry), "");
static_assert(offsetof(SgCtx, fibEntryDyn) == offsetof(FwFwdCtx, fibEntryDyn), "");
static_assert(offsetof(SgCtx, pitEntry) == offsetof(FwFwdCtx, pitEntry), "");
static_assert(sizeof(SgCtx) == offsetof(FwFwdCtx, endofSgCtx), "");

static const size_t FwFwd_OffsetofQueue[PktMax] = {
  SIZE_MAX,
  offsetof(FwFwd, queueI),
  offsetof(FwFwd, queueD),
  offsetof(FwFwd, queueN),
};

typedef void (*FwFwd_RxFunc)(FwFwd* fwd, FwFwdCtx* ctx);
static const FwFwd_RxFunc FwFwd_RxFuncs[PktMax] = {
  NULL,
  FwFwd_RxInterest,
  FwFwd_RxData,
  FwFwd_RxNack,
};

static __rte_always_inline void
FwFwd_RxByType(FwFwd* fwd, PktType pktType)
{
  NDNDPDK_ASSERT(pktType < PktMax);
  TscTime now = rte_get_tsc_cycles();
  PktQueue* q = RTE_PTR_ADD(fwd, FwFwd_OffsetofQueue[pktType]);
  struct rte_mbuf* pkts[MaxBurstSize];
  PktQueuePopResult pop = PktQueue_Pop(q, pkts, RTE_DIM(pkts), now);
  if (unlikely(pop.drop)) {
    Packet_GetLpL3Hdr(Packet_FromMbuf(pkts[0]))->congMark = 1;
  }
  for (uint32_t i = 0; i < pop.count; ++i) {
    FwFwdCtx ctx = {
      .fwd = fwd,
      .pkt = pkts[i],
    };
    ctx.rxFace = ctx.pkt->port;
    ctx.rxTime = ctx.pkt->timestamp;
    ctx.rxToken = Packet_GetLpL3Hdr(ctx.npkt)->pitToken;
    ctx.eventKind = (SgEvent)pktType;

    TscDuration timeSinceRx = now - ctx.rxTime;
    RunningStat_Push1(&fwd->latencyStat, timeSinceRx);

    (*FwFwd_RxFuncs[pktType])(fwd, &ctx);
  }
}

int
FwFwd_Run(FwFwd* fwd)
{
  rcu_register_thread();
  ZF_LOGI("Run(%" PRIu8 ") fwd=%p fib=%p pit=%p cs=%p crypto=%p", fwd->id, fwd, fwd->fib, fwd->pit,
          fwd->cs, fwd->crypto);

  fwd->sgGlobal.tscHz = rte_get_tsc_hz();
  Pit_SetSgTimerCb(fwd->pit, SgTriggerTimer, fwd);

  while (ThreadStopFlag_ShouldContinue(&fwd->stop)) {
    rcu_quiescent_state();
    Pit_TriggerTimers(fwd->pit);

    FwFwd_RxByType(fwd, PktInterest);
    FwFwd_RxByType(fwd, PktData);
    FwFwd_RxByType(fwd, PktNack);
  }

  ZF_LOGI("Stop(%" PRIu8 ")", fwd->id);
  rcu_unregister_thread();
  return 0;
}
