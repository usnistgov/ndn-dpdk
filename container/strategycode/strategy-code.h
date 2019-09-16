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

void
StrategyCode_Ref(StrategyCode* sc);

void
StrategyCode_Unref(StrategyCode* sc);

const struct ebpf_insn*
StrategyCode_GetEmptyProgram_(uint32_t* nInsn);

static __rte_always_inline struct rte_bpf*
rte_bpf_elf_load_(const struct rte_bpf_prm* prm,
                  const char* fname,
                  const char* sname)
{
#pragma GCC diagnostic push
#pragma GCC diagnostic ignored "-Wdeprecated-declarations"
  return rte_bpf_elf_load(prm, fname, sname);
#pragma GCC diagnostic pop
}

static __rte_always_inline struct rte_bpf*
rte_bpf_load_(const struct rte_bpf_prm* prm)
{
#pragma GCC diagnostic push
#pragma GCC diagnostic ignored "-Wdeprecated-declarations"
  return rte_bpf_load(prm);
#pragma GCC diagnostic pop
}

static __rte_always_inline void
rte_bpf_destroy_(struct rte_bpf* bpf)
{
#pragma GCC diagnostic push
#pragma GCC diagnostic ignored "-Wdeprecated-declarations"
  return rte_bpf_destroy(bpf);
#pragma GCC diagnostic pop
}

static __rte_always_inline int
rte_bpf_get_jit_(const struct rte_bpf* bpf, struct rte_bpf_jit* jit)
{
#pragma GCC diagnostic push
#pragma GCC diagnostic ignored "-Wdeprecated-declarations"
  return rte_bpf_get_jit(bpf, jit);
#pragma GCC diagnostic pop
}

#endif // NDN_DPDK_CONTAINER_STRATEGYCODE_STRATEGY_CODE_H
