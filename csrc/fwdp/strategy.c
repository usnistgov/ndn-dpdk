#include "strategy.h"

#include "../core/logger.h"

N_LOG_INIT(FwFwd);

void
SgTriggerTimer(Pit* pit, PitEntry* pitEntry, void* fwd0)
{
  FwFwd* fwd = fwd0;
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
    rcu_read_unlock();
    return;
  }

  // invoke strategy
  N_LOGD("Timer invoke sgtimer-at=%p fib-entry=%p sg-id=%d", pitEntry, ctx.fibEntry,
         ctx.fibEntry->strategy->id);
  uint64_t res = SgInvoke(ctx.fibEntry->strategy, &ctx);
  N_LOGD("^ sg-res=%" PRIu64 " sg-forwarded=%d", res, ctx.nForwarded);

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
SgGetXsyms(int* nXsyms)
{
  static const struct rte_bpf_xsym xsyms[] =
    { {
        .name = "SgSetTimer",
        .type = RTE_BPF_XTYPE_FUNC,
        .func =
          {
            .val = (void*)SgSetTimer,
            .nb_args = 2,
            .args =
              {
                [0] =
                  {
                    .type = RTE_BPF_ARG_PTR,
                    .size = sizeof(SgCtx),
                  },
                [1] =
                  {
                    .type = RTE_BPF_ARG_RAW,
                  },
              },
          },
      },
      {
        .name = "SgForwardInterest",
        .type = RTE_BPF_XTYPE_FUNC,
        .func =
          {
            .val = (void*)SgForwardInterest,
            .nb_args = 2,
            .args =
              {
                [0] =
                  {
                    .type = RTE_BPF_ARG_PTR,
                    .size = sizeof(SgCtx),
                  },
                [1] =
                  {
                    .type = RTE_BPF_ARG_RAW,
                  },
              },
          },
      },
      { .name = "SgReturnNacks",
        .type = RTE_BPF_XTYPE_FUNC,
        .func = {
          .val = (void*)SgReturnNacks,
          .nb_args = 2,
          .args =
            {
              [0] =
                {
                  .type = RTE_BPF_ARG_PTR,
                  .size = sizeof(SgCtx),
                },
              [1] =
                {
                  .type = RTE_BPF_ARG_RAW,
                },
            },
        } } };
  *nXsyms = RTE_DIM(xsyms);
  return xsyms;
}
