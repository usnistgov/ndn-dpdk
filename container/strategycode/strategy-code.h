#ifndef NDN_DPDK_CONTAINER_STRATEGYCODE_STRATEGY_CODE_H
#define NDN_DPDK_CONTAINER_STRATEGYCODE_STRATEGY_CODE_H

/// \file

#include "../../core/common.h"
#include <rte_bpf.h>

typedef uint64_t (*StrategyCodeFunc)(void*, size_t);

/** \brief BPF program of a forwarding strategy.
 */
typedef struct StrategyCode
{
  char* name;           ///< descriptive name
  struct rte_bpf* bpf;  ///< BPF execution context
  StrategyCodeFunc jit; ///< JIT-compiled strategy function
  int id;               ///< identifier
  atomic_int nRefs;     ///< how many FibEntry* references this
} StrategyCode;

/** \brief Execute strategy BPF program.
 *  \param arg argument to BPF program.
 *  \param sizeofArg sizeof(*arg)
 */
static uint64_t
StrategyCode_Execute(StrategyCode* sc, void* arg, size_t sizeofArg)
{
  return (*sc->jit)(arg, sizeofArg);
}

void StrategyCode_Ref(StrategyCode* sc);

void StrategyCode_Unref(StrategyCode* sc);

const struct ebpf_insn* __StrategyCode_GetEmptyProgram(uint32_t* nInsn);

#endif // NDN_DPDK_CONTAINER_STRATEGYCODE_STRATEGY_CODE_H
