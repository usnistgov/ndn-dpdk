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

/** \brief Register BPF-CALLable functions.
 *  \return number of errors.
 */
int SgRegisterFuncs(struct ubpf_vm* vm);

/** \brief Invoke the strategy.
 */
static uint64_t
SgInvoke(StrategyCode* strategy, SgContext* ctx)
{
  return (*strategy->jit)(ctx, sizeof(SgContext));
}

#endif // NDN_DPDK_APP_FWDP_STRATEGY_H
