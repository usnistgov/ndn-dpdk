#ifndef NDNDPDK_STRATEGYCODE_STRATEGY_CODE_H
#define NDNDPDK_STRATEGYCODE_STRATEGY_CODE_H

/** @file */

#include "../core/common.h"
#include <rte_bpf.h>

typedef uint64_t (*StrategyCodeFunc)(void*, size_t);

/** @brief BPF program in a forwarding strategy. */
typedef struct StrategyCodeProg
{
  struct rte_bpf* bpf;  ///< BPF execution context
  StrategyCodeFunc jit; ///< JIT-compiled function
} StrategyCodeProg;

__attribute__((nonnull)) static inline uint64_t
StrategyCodeProg_Run(StrategyCodeProg prog, void* arg, size_t sizeofArg)
{
  return prog.jit(arg, sizeofArg);
}

/** @brief Forwarding strategy BPF programs. */
typedef struct StrategyCode
{
  StrategyCodeProg main; ///< dataplane BPF program
  uintptr_t goHandle;    ///< cgo.Handle reference of Go *strategycode.Strategy
  int id;                ///< strategy ID
  atomic_int nRefs;      ///< how many FibEntry* reference this
} StrategyCode;

/**
 * @brief Run the forwarding strategy dataplane BPF program.
 * @param arg argument to BPF program.
 * @param sizeofArg sizeof(*arg)
 */
__attribute__((nonnull)) static inline uint64_t
StrategyCode_Run(StrategyCode* sc, void* arg, size_t sizeofArg)
{
  return StrategyCodeProg_Run(sc->main, arg, sizeofArg);
}

__attribute__((nonnull)) void
StrategyCode_Ref(StrategyCode* sc);

__attribute__((nonnull)) void
StrategyCode_Unref(StrategyCode* sc);

typedef void (*StrategyCode_FreeFunc)(uintptr_t goHandle);
extern StrategyCode_FreeFunc StrategyCode_Free;

typedef struct SgCtx SgCtx;
typedef bool (*StrategyCode_GetJSONFunc)(SgCtx* ctx, const char* path, int index, int64_t* dst);
extern StrategyCode_GetJSONFunc StrategyCode_GetJSON;

__attribute__((nonnull, returns_nonnull)) const struct rte_bpf_xsym*
SgInitGetXsyms(uint32_t* nXsyms);

#endif // NDNDPDK_STRATEGYCODE_STRATEGY_CODE_H
