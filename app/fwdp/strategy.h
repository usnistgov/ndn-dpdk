#ifndef NDN_DPDK_APP_FWDP_STRATEGY_H
#define NDN_DPDK_APP_FWDP_STRATEGY_H

/// \file

#include "../../strategy/api.h"
#include "fwd.h"

/** \brief Context of strategy invocation.
 *
 *  This is an extension of \c SgCtx, to include fields needed by forwarding
 *  but inaccessible in strategy.
 */
typedef struct SgContext
{
  SgCtx inner;

  FwFwd* fwd;

  TscTime rxTime;   // SGEVT_INTEREST and SGEVT_NACK only
  uint32_t dnNonce; // SGEVT_INTEREST and SGEVT_NACK only
  int nForwarded;   // SGEVT_INTEREST and SGEVT_NACK only
} SgContext;

/** \brief Obtain external symbols available to strategy eBPF program.
 */
const struct rte_bpf_xsym* SgGetXsyms(int* nXsyms);

/** \brief Invoke the strategy.
 */
static uint64_t
SgInvoke(StrategyCode* strategy, SgContext* ctx)
{
  return StrategyCode_Execute(strategy, ctx, sizeof(SgCtx));
}

#endif // NDN_DPDK_APP_FWDP_STRATEGY_H
