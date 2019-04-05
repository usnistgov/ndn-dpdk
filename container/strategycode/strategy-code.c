#include "strategy-code.h"

void
StrategyCode_Ref(StrategyCode* sc)
{
  assert(sc->bpf != NULL);
  assert(sc->jit != NULL);
  atomic_fetch_add_explicit(&sc->nRefs, 1, memory_order_acq_rel);
}

void
StrategyCode_Unref(StrategyCode* sc)
{
  int oldNRefs = atomic_fetch_sub_explicit(&sc->nRefs, 1, memory_order_acq_rel);
  assert(oldNRefs > 0);
}

const struct ebpf_insn*
__StrategyCode_GetEmptyProgram(uint32_t* nInsn)
{
  static const struct ebpf_insn program[] = {
    {
      0, .code = BPF_ALU | EBPF_MOV | BPF_K, .dst_reg = EBPF_REG_0, .imm = 0,
    },
    {
      0, .code = BPF_JMP | EBPF_EXIT,
    },
  };
  *nInsn = RTE_DIM(program);
  return program;
}
