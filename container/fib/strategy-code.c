#include "strategy-code.h"
#include "fib.h"

static_assert(sizeof(StrategyCode) <= sizeof(FibEntry), "");

StrategyCode*
StrategyCode_Alloc(Fib* fib)
{
  StrategyCode* sc = NULL;
  int res = rte_mempool_get(Tsht_ToMempool(Fib_ToTsht(fib)), (void**)&sc);
  if (unlikely(res != 0)) {
    return NULL;
  }
  memset(sc, 0, sizeof(StrategyCode));
  return sc;
}

void
StrategyCode_Ref(StrategyCode* sc)
{
  assert(sc->vm != NULL);
  assert(sc->jit != NULL);
  ++sc->nRefs;
}

void
StrategyCode_Unref(StrategyCode* sc)
{
  assert(sc->nRefs > 0);
  --sc->nRefs;
  if (sc->nRefs > 0) {
    return;
  }
  ubpf_destroy(sc->vm);
  rte_mempool_put(rte_mempool_from_obj(sc), sc);
}
