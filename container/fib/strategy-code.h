#ifndef NDN_DPDK_APP_FWDP_STRATEGY_CODE_H
#define NDN_DPDK_APP_FWDP_STRATEGY_CODE_H

/// \file

#include "../../core/common.h"
#include <ubpf.h>

typedef struct Fib Fib;

typedef struct StrategyCode
{
  struct ubpf_vm* vm;
  ubpf_jit_fn jit;
  int id;
  int nRefs; ///< how many FibEntry* references this
} StrategyCode;

StrategyCode* StrategyCode_Alloc(Fib* fib);

void StrategyCode_Ref(StrategyCode* sc);

void StrategyCode_Unref(StrategyCode* sc);

#endif // NDN_DPDK_APP_FWDP_STRATEGY_CODE_H
