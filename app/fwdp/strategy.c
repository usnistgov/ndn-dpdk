#include "strategy.h"

#include "../../core/logger.h"

INIT_ZF_LOG(FwFwd);

void
SgTriggerTimer(Pit* pit, PitEntry* pitEntry, void* fwd0)
{
  FwFwd* fwd = (FwFwd*)fwd0;

  // find FIB entry
  rcu_read_lock();
  const FibEntry* fibEntry = PitEntry_FindFibEntry(pitEntry, fwd->fib);
  if (unlikely(fibEntry == NULL)) {
    ZF_LOGD("sgtimer-at=%p drop=no-FIB-match", pitEntry);
    rcu_read_unlock();
    return;
  }

  // invoke strategy
  ZF_LOGD("sgtimer-at=%p fib-entry=%p sg-id=%d",
          pitEntry,
          fibEntry,
          fibEntry->strategy->id);
  SgContext sgCtx = { 0 };
  sgCtx.inner.eventKind = SGEVT_TIMER;
  sgCtx.inner.fibEntry = (const SgFibEntry*)fibEntry;
  sgCtx.inner.pitEntry = (SgPitEntry*)pitEntry;
  sgCtx.fwd = fwd;
  uint64_t res = SgInvoke(fibEntry->strategy, &sgCtx);
  ZF_LOGD("^ sg-res=%" PRIu64 " sg-forwarded=%d", res, sgCtx.nForwarded);
  rcu_read_unlock();
}

bool
SgSetTimer(SgCtx* ctx0, int afterMillis)
{
  TscDuration after = TscDuration_FromMillis(afterMillis);
  SgContext* ctx = (SgContext*)ctx0;
  PitEntry* pitEntry = (PitEntry*)ctx->inner.pitEntry;
  bool ok = PitEntry_SetSgTimer(pitEntry, ctx->fwd->pit, after);
  ZF_LOGD("^ sgtimer-after=%dms %s", afterMillis, ok ? "OK" : "FAIL");
  return ok;
}

const struct rte_bpf_xsym*
SgGetXsyms(int* nXsyms)
{
  static const struct rte_bpf_xsym xsyms[] =
    { {
        0,
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
        0,
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
      { 0,
        .name = "SgReturnNacks",
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
