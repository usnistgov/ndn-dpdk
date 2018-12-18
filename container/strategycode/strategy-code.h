#ifndef NDN_DPDK_CONTAINER_STRATEGYCODE_STRATEGY_CODE_H
#define NDN_DPDK_CONTAINER_STRATEGYCODE_STRATEGY_CODE_H

/// \file

#include "../../core/common.h"
#include <ubpf.h>

/** \brief BPF program of a forwarding strategy.
 */
typedef struct StrategyCode
{
  char* name;         ///< descriptive name
  struct ubpf_vm* vm; ///< BPF virtual machine
  ubpf_jit_fn jit;    ///< JIT-compiled strategy function
  int id;             ///< identifier
  atomic_int nRefs;   ///< how many FibEntry* references this
} StrategyCode;

void StrategyCode_Ref(StrategyCode* sc);

void StrategyCode_Unref(StrategyCode* sc);

#endif // NDN_DPDK_CONTAINER_STRATEGYCODE_STRATEGY_CODE_H
