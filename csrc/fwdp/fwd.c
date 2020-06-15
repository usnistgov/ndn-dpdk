#include "fwd.h"
#include "strategy.h"

#include "../core/logger.h"

INIT_ZF_LOG(FwFwd);

static_assert((int)SGEVT_INTEREST == (int)L3PktType_Interest, "");
static_assert((int)SGEVT_DATA == (int)L3PktType_Data, "");
static_assert((int)SGEVT_NACK == (int)L3PktType_Nack, "");
static_assert(offsetof(SgCtx, global) == offsetof(FwFwdCtx, fwd), "");
static_assert(offsetof(FwFwd, sgGlobal) == 0, "");
static_assert(offsetof(SgCtx, now) == offsetof(FwFwdCtx, rxTime), "");
static_assert(offsetof(SgCtx, eventKind) == offsetof(FwFwdCtx, eventKind), "");
static_assert(offsetof(SgCtx, nhFlt) == offsetof(FwFwdCtx, nhFlt), "");
static_assert(offsetof(SgCtx, pkt) == offsetof(FwFwdCtx, pkt), "");
static_assert(offsetof(SgCtx, fibEntry) == offsetof(FwFwdCtx, fibEntry), "");
static_assert(offsetof(SgCtx, pitEntry) == offsetof(FwFwdCtx, pitEntry), "");
static_assert(sizeof(SgCtx) == offsetof(FwFwdCtx, endofSgCtx), "");

typedef void (*FwFwd_RxFunc)(FwFwd* fwd, FwFwdCtx* ctx);
static const FwFwd_RxFunc FwFwd_RxFuncs[L3PktType_MAX] = {
  NULL,
  FwFwd_RxInterest,
  FwFwd_RxData,
  FwFwd_RxNack,
};

static __rte_always_inline void
FwFwd_RxByType(FwFwd* fwd, L3PktType l3type)
{
  TscTime now = rte_get_tsc_cycles();
  PktQueue* q = RTE_PTR_ADD(fwd, FwFwd_OffsetofQueue[l3type]);
  struct rte_mbuf* pkts[PKTQUEUE_BURST_SIZE_MAX];
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
    ctx.eventKind = (SgEvent)l3type;

    TscDuration timeSinceRx = now - ctx.rxTime;
    RunningStat_Push1(&fwd->latencyStat, timeSinceRx);

    (*FwFwd_RxFuncs[l3type])(fwd, &ctx);
  }
}

void
FwFwd_Run(FwFwd* fwd)
{
  ZF_LOGI("fwdId=%" PRIu8 " fwd=%p fib=%p pit+cs=%p crypto=%p",
          fwd->id,
          fwd,
          fwd->fib,
          fwd->pcct,
          fwd->crypto);

  fwd->sgGlobal.tscHz = rte_get_tsc_hz();
  Pit_SetSgTimerCb(fwd->pit, SgTriggerTimer, fwd);

  while (ThreadStopFlag_ShouldContinue(&fwd->stop)) {
    rcu_quiescent_state();
    Pit_TriggerTimers(fwd->pit);

    FwFwd_RxByType(fwd, L3PktType_Interest);
    FwFwd_RxByType(fwd, L3PktType_Data);
    FwFwd_RxByType(fwd, L3PktType_Nack);
  }

  ZF_LOGI("fwdId=%" PRIu8 " STOP", fwd->id);
}
