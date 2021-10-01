#ifndef NDNDPDK_STRATEGYCODE_STRATEGY_CODE_H
#define NDNDPDK_STRATEGYCODE_STRATEGY_CODE_H

/** @file */

#include "../core/common.h"
#include <rte_bpf.h>

typedef uint64_t (*StrategyCodeFunc)(void*, size_t);

/** @brief BPF program of a forwarding strategy. */
typedef struct StrategyCode
{
  char* name;           ///< descriptive name
  struct rte_bpf* bpf;  ///< BPF execution context
  StrategyCodeFunc jit; ///< JIT-compiled strategy function
  int id;               ///< identifier
  atomic_int nRefs;     ///< how many FibEntry* reference this
} StrategyCode;

/**
 * @brief Run the strategy's BPF program.
 * @param arg argument to BPF program.
 * @param sizeofArg sizeof(*arg)
 */
__attribute__((nonnull)) static inline uint64_t
StrategyCode_Run(StrategyCode* sc, void* arg, size_t sizeofArg)
{
  return (*sc->jit)(arg, sizeofArg);
}

__attribute__((nonnull)) void
StrategyCode_Ref(StrategyCode* sc);

__attribute__((nonnull)) void
StrategyCode_Unref(StrategyCode* sc);

__attribute__((nonnull, returns_nonnull)) const struct ebpf_insn*
StrategyCode_GetEmptyProgram_(uint32_t* nInsn);

#endif // NDNDPDK_STRATEGYCODE_STRATEGY_CODE_H
