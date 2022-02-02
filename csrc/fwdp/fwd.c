#include "fwd.h"
#include "strategy.h"

#include "../core/logger.h"

N_LOG_INIT(FwFwd);

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

typedef void (*RxFunc)(FwFwd* fwd, FwFwdCtx* ctx);

__attribute__((nonnull)) static inline uint32_t
FwFwd_RxBurst(FwFwd* fwd, PktType pktType, PktQueue* q, RxFunc process)
{
  TscTime now = rte_get_tsc_cycles();
  struct rte_mbuf* pkts[MaxBurstSize];
  PktQueuePopResult pop = PktQueue_Pop(q, pkts, RTE_DIM(pkts), now);
  if (unlikely(pop.drop)) {
    Packet_GetLpL3Hdr(Packet_FromMbuf(pkts[0]))->congMark = 1;
  }

  for (uint32_t i = 0; i < pop.count; ++i) {
    FwFwdCtx ctx = {
      .fwd = fwd,
      .eventKind = (SgEvent)pktType,
      .pkt = pkts[i],
    };
    ctx.rxFace = ctx.pkt->port;
    ctx.rxTime = Mbuf_GetTimestamp(ctx.pkt);
    ctx.rxToken = Packet_GetLpL3Hdr(ctx.npkt)->pitToken;

    TscDuration timeSinceRx = now - ctx.rxTime;
    RunningStat_Push1(&fwd->latencyStat, timeSinceRx);

    process(fwd, &ctx);
  }

  return pop.count;
}

int
FwFwd_Run(FwFwd* fwd)
{
  rcu_register_thread();
  N_LOGI("Run fwd-id=%" PRIu8 " fwd=%p fib=%p pit=%p cs=%p crypto=%p", fwd->id, fwd, fwd->fib,
         fwd->pit, fwd->cs, fwd->cryptoHelper);

  fwd->sgGlobal.tscHz = TscHz;
  Pit_SetSgTimerCb(fwd->pit, SgTriggerTimer, fwd);

  uint32_t nProcessed = 0;
  while (ThreadCtrl_Continue(fwd->ctrl, nProcessed)) {
    rcu_quiescent_state();
    Pit_TriggerTimers(fwd->pit);

    nProcessed += FwFwd_RxBurst(fwd, PktInterest, &fwd->queueI, FwFwd_RxInterest);
    nProcessed += FwFwd_RxBurst(fwd, PktData, &fwd->queueD, FwFwd_RxData);
    nProcessed += FwFwd_RxBurst(fwd, PktNack, &fwd->queueN, FwFwd_RxNack);
  }

  N_LOGI("Stop fwd-id=%" PRIu8, fwd->id);
  rcu_unregister_thread();
  return 0;
}
