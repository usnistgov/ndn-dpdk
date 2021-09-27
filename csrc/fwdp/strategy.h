#ifndef NDNDPDK_FWDP_STRATEGY_H
#define NDNDPDK_FWDP_STRATEGY_H

/** @file */

#include "../strategyapi/api.h"
#include "fwd.h"

/** @brief Obtain external symbols available to strategy eBPF program. */
const struct rte_bpf_xsym*
SgGetXsyms(int* nXsyms);

__attribute__((nonnull)) void
SgTriggerTimer(Pit* pit, PitEntry* pitEntry, void* fwd0);

/** @brief Invoke the strategy. */
static inline uint64_t
SgInvoke(StrategyCode* strategy, FwFwdCtx* ctx)
{
  return StrategyCode_Run(strategy, ctx, sizeof(SgCtx));
}

#endif // NDNDPDK_FWDP_STRATEGY_H
