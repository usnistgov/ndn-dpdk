#ifndef NDNDPDK_FWDP_STRATEGY_H
#define NDNDPDK_FWDP_STRATEGY_H

/** @file */

#include "../strategyapi/api.h"
#include "fwd.h"

/** @brief Obtain external symbols available to strategy eBPF programs. */
__attribute__((nonnull, returns_nonnull)) const struct rte_bpf_xsym*
SgGetXsyms(int* nXsyms);

__attribute__((nonnull)) void
SgTriggerTimer(Pit* pit, PitEntry* pitEntry, uintptr_t fwd0);

/** @brief Invoke the strategy. */
__attribute__((nonnull)) static inline uint64_t
SgInvoke(StrategyCode* strategy, FwFwdCtx* ctx)
{
  return StrategyCode_Run(strategy, ctx, sizeof(SgCtx));
}

#endif // NDNDPDK_FWDP_STRATEGY_H
