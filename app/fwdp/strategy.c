#include "strategy.h"

int
SgRegisterFuncs(struct ubpf_vm* vm)
{
  unsigned int index = 0;
  int nErrors = 0;
  nErrors -= ubpf_register(vm, ++index, "ForwardInterest", Sg_ForwardInterest);
  return nErrors;
}
