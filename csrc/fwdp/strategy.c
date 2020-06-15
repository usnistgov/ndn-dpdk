#include "strategy.h"

#include "../core/logger.h"

INIT_ZF_LOG(FwFwd);

void
SgTriggerTimer(Pit* pit, PitEntry* pitEntry, void* fwd0)
{
  FwFwdCtx ctx = {
    .rxTime = rte_get_tsc_cycles(),
    .fwd = (FwFwd*)fwd0,
    .eventKind = SGEVT_TIMER,
    .pitEntry = pitEntry,
  };

  // find FIB entry
  rcu_read_lock();
  ctx.fibEntry = PitEntry_FindFibEntry(pitEntry, ctx.fwd->fib);
  if (unlikely(ctx.fibEntry == NULL)) {
    ZF_LOGD("sgtimer-at=%p drop=no-FIB-match", pitEntry);
    rcu_read_unlock();
    return;
  }

  // invoke strategy
  ZF_LOGD("sgtimer-at=%p fib-entry=%p sg-id=%d",
          pitEntry,
          ctx.fibEntry,
          ctx.fibEntry->strategy->id);
  uint64_t res = SgInvoke(ctx.fibEntry->strategy, &ctx);
  ZF_LOGD("^ sg-res=%" PRIu64 " sg-forwarded=%d", res, ctx.nForwarded);

  FwFwd_NULLize(ctx.fibEntry); // fibEntry is inaccessible upon RCU unlock
  rcu_read_unlock();
}

bool
SgSetTimer(SgCtx* ctx0, TscDuration after)
{
  FwFwdCtx* ctx = (FwFwdCtx*)ctx0;
  bool ok = PitEntry_SetSgTimer(ctx->pitEntry, ctx->fwd->pit, after);
  ZF_LOGD("^ sgtimer-after=%" PRId64 " %s", after, ok ? "OK" : "FAIL");
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
