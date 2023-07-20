#include "entry.h"
#include "../strategyapi/api.h"

static_assert(offsetof(SgCtx, global) == offsetof(FibSgInitCtx, global), "");
static_assert(offsetof(SgCtx, now) == offsetof(FibSgInitCtx, now), "");
static_assert(offsetof(SgCtx, fibEntry) == offsetof(FibSgInitCtx, entry), "");
static_assert(offsetof(SgCtx, fibEntryDyn) == offsetof(FibSgInitCtx, dyn), "");
static_assert(sizeof(SgCtx) <= offsetof(FibSgInitCtx, goHandle), "");

__attribute__((nonnull)) static void
FibEntry_RcuFree(struct rcu_head* rcuhead) {
  FibEntry* entry = container_of(rcuhead, FibEntry, rcuhead);
  if (entry->height == 0) {
    StrategyCode_Unref(entry->strategy);
  }
  rte_mempool_put(rte_mempool_from_obj(entry), entry);
}

void
FibEntry_DeferredFree(FibEntry* entry) {
  call_rcu(&entry->rcuhead, FibEntry_RcuFree);
}
