#include "fwd.h"
#include "strategy.h"

#include "../../core/logger.h"

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

#define FW_FWD_BURST_SIZE 16

typedef void (*FwFwd_RxFunc)(FwFwd* fwd, FwFwdCtx* ctx);
static const FwFwd_RxFunc FwFwd_RxFuncs[L3PktType_MAX] = {
  NULL,
  FwFwd_RxInterest,
  FwFwd_RxData,
  FwFwd_RxNack,
};

void
FwFwd_Run(FwFwd* fwd)
{
  ZF_LOGI("fwdId=%" PRIu8 " fwd=%p queue=%p fib=%p pit+cs=%p crypto=%p",
          fwd->id,
          fwd,
          fwd->queue,
          fwd->fib,
          fwd->pcct,
          fwd->crypto);

  fwd->sgGlobal.tscHz = rte_get_tsc_hz();
  Pit_SetSgTimerCb(fwd->pit, SgTriggerTimer, fwd);

  Packet* npkts[FW_FWD_BURST_SIZE];
  while (ThreadStopFlag_ShouldContinue(&fwd->stop)) {
    rcu_quiescent_state();
    Pit_TriggerTimers(fwd->pit);

    unsigned count = rte_ring_dequeue_burst(
      fwd->queue, (void**)npkts, FW_FWD_BURST_SIZE, NULL);
    TscTime now = rte_get_tsc_cycles();
    for (unsigned i = 0; i < count; ++i) {
      FwFwdCtx ctx = {
        .fwd = fwd,
        .npkt = npkts[i],
      };
      ctx.rxFace = ctx.pkt->port;
      ctx.rxTime = ctx.pkt->timestamp;
      ctx.rxToken = Packet_GetLpL3Hdr(ctx.npkt)->pitToken;
      ctx.eventKind = (SgEvent)Packet_GetL3PktType(ctx.npkt);

      TscDuration timeSinceRx = now - ctx.rxTime;
      RunningStat_Push1(&fwd->latencyStat, timeSinceRx);

      (*FwFwd_RxFuncs[ctx.eventKind])(fwd, &ctx);
    }
  }

  ZF_LOGI("fwdId=%" PRIu8 " STOP", fwd->id);
}
