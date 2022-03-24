#include "strategy.h"

#include "../core/logger.h"

N_LOG_INIT(FwFwd);

uint32_t
SgRandInt(SgCtx* ctx0, uint32_t max)
{
  FwFwdCtx* ctx = (FwFwdCtx*)ctx0;
  return pcg32_boundedrand_r(&ctx->fwd->sgRng, max);
}

void
SgTriggerTimer(Pit* pit, PitEntry* pitEntry, uintptr_t fwd0)
{
  FwFwd* fwd = (FwFwd*)fwd0;
  FwFwdCtx ctx = {
    .rxTime = rte_get_tsc_cycles(),
    .fwd = fwd,
    .eventKind = SGEVT_TIMER,
    .pitEntry = pitEntry,
  };

  // find FIB entry
  rcu_read_lock();
  FwFwdCtx_SetFibEntry(&ctx, PitEntry_FindFibEntry(pitEntry, ctx.fwd->fib));
  if (unlikely(ctx.fibEntry == NULL)) {
    N_LOGD("Timer no-FIB-match sgtimer-at=%p", pitEntry);
    goto FINISH;
  }

  // invoke strategy
  N_LOGD("Timer invoke sgtimer-at=%p fib-entry=%p sg-id=%d", pitEntry, ctx.fibEntry,
         ctx.fibEntry->strategy->id);
  uint64_t res = SgInvoke(ctx.fibEntry->strategy, &ctx);
  N_LOGD("^ sg-res=%" PRIu64 " sg-forwarded=%d", res, ctx.nForwarded);

FINISH:
  NULLize(ctx.fibEntry); // fibEntry is inaccessible upon RCU unlock
  rcu_read_unlock();
}

bool
SgSetTimer(SgCtx* ctx0, TscDuration after)
{
  FwFwdCtx* ctx = (FwFwdCtx*)ctx0;
  bool ok = PitEntry_SetSgTimer(ctx->pitEntry, ctx->fwd->pit, after);
  N_LOGD("^ sgtimer-after=%" PRId64 " %s", after, ok ? "OK" : "FAIL");
  return ok;
}

const struct rte_bpf_xsym*
SgGetXsyms(uint32_t* nXsyms)
{
  static const struct rte_bpf_xsym xsyms[] = {
    {
      .name = "SgRandInt",
      .type = RTE_BPF_XTYPE_FUNC,
      .func = {
        .val = (void*)SgRandInt,
        .nb_args = 2,
        .args = {
          { .type = RTE_BPF_ARG_PTR, .size = sizeof(SgCtx) },
          { .type = RTE_BPF_ARG_RAW },
        },
        .ret = { .type = RTE_BPF_ARG_RAW },
      },
    },
    {
      .name = "SgSetTimer",
      .type = RTE_BPF_XTYPE_FUNC,
      .func = {
        .val = (void*)SgSetTimer,
        .nb_args = 2,
        .args = {
          { .type = RTE_BPF_ARG_PTR, .size = sizeof(SgCtx) },
          { .type = RTE_BPF_ARG_RAW },
        },
        .ret = { .type = RTE_BPF_ARG_UNDEF },
      },
    },
    {
      .name = "SgForwardInterest",
      .type = RTE_BPF_XTYPE_FUNC,
      .func = {
        .val = (void*)SgForwardInterest,
        .nb_args = 2,
        .args = {
          { .type = RTE_BPF_ARG_PTR, .size = sizeof(SgCtx) },
          { .type = RTE_BPF_ARG_RAW },
        },
        .ret = { .type = RTE_BPF_ARG_RAW },
      },
    },
    {
      .name = "SgReturnNacks",
      .type = RTE_BPF_XTYPE_FUNC,
      .func = {
        .val = (void*)SgReturnNacks,
        .nb_args = 2,
        .args = {
          { .type = RTE_BPF_ARG_PTR, .size = sizeof(SgCtx) },
          { .type = RTE_BPF_ARG_RAW },
        },
        .ret = { .type = RTE_BPF_ARG_UNDEF },
      },
    },
  };
  *nXsyms = RTE_DIM(xsyms);
  return xsyms;
}
