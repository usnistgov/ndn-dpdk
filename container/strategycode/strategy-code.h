#ifndef NDN_DPDK_CONTAINER_STRATEGY_CODE_STRATEGY_CODE_H
#define NDN_DPDK_CONTAINER_STRATEGY_CODE_STRATEGY_CODE_H

/// \file

#include "../../core/common.h"
#include <ubpf.h>

typedef struct StrategyCode
{
  struct ubpf_vm* vm;
  ubpf_jit_fn jit;
  int id;
  atomic_int nRefs; ///< how many FibEntry* references this
} StrategyCode;

void StrategyCode_Ref(StrategyCode* sc);

void StrategyCode_Unref(StrategyCode* sc);

#endif // NDN_DPDK_CONTAINER_STRATEGY_CODE_STRATEGY_CODE_H
