#include "strategy-code.h"

void
StrategyCode_Ref(StrategyCode* sc)
{
  assert(sc->vm != NULL);
  assert(sc->jit != NULL);
  atomic_fetch_add_explicit(&sc->nRefs, 1, memory_order_acq_rel);
}

void
StrategyCode_Unref(StrategyCode* sc)
{
  int oldNRefs = atomic_fetch_sub_explicit(&sc->nRefs, 1, memory_order_acq_rel);
  assert(oldNRefs > 0);
}
