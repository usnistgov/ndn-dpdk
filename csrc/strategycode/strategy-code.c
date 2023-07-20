#include "strategy-code.h"
#include "../strategyapi/api.h"

StrategyCode_FreeFunc StrategyCode_Free;
StrategyCode_GetJSONFunc StrategyCode_GetJSON;

void
StrategyCode_Ref(StrategyCode* sc) {
  NDNDPDK_ASSERT(sc->main.bpf != NULL);
  NDNDPDK_ASSERT(sc->main.jit != NULL);
  NDNDPDK_ASSERT(sc->id != 0);
  NDNDPDK_ASSERT(sc->goHandle != 0);
  atomic_fetch_add_explicit(&sc->nRefs, 1, memory_order_acq_rel);
}

void
StrategyCode_Unref(StrategyCode* sc) {
  int oldNRefs = atomic_fetch_sub_explicit(&sc->nRefs, 1, memory_order_acq_rel);
  NDNDPDK_ASSERT(oldNRefs > 0);
  if (oldNRefs > 1) {
    return;
  }

  StrategyCode_Free(sc->goHandle);
}

bool
SgGetJSON(SgCtx* ctx, const char* path, int index, int64_t* dst) {
  NDNDPDK_ASSERT(StrategyCode_GetJSON != NULL);
  return StrategyCode_GetJSON(ctx, path, index, dst);
}

const struct rte_bpf_xsym*
SgInitGetXsyms(uint32_t* nXsyms) {
  static const struct rte_bpf_xsym xsyms[] = {
    {
      .name = "SgGetJSON",
      .type = RTE_BPF_XTYPE_FUNC,
      .func =
        {
          .val = (void*)SgGetJSON,
          .nb_args = 4,
          .args =
            {
              {.type = RTE_BPF_ARG_PTR, .size = sizeof(SgCtx)},
              {.type = RTE_BPF_ARG_RAW},
              {.type = RTE_BPF_ARG_RAW},
              {.type = RTE_BPF_ARG_PTR, .size = sizeof(int64_t)},
            },
          .ret = {.type = RTE_BPF_ARG_RAW},
        },
    },
  };
  *nXsyms = RTE_DIM(xsyms);
  return xsyms;
}
