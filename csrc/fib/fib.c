#include "fib.h"

__attribute__((nonnull)) static int // bool
Fib_LookupMatch_(struct cds_lfht_node* lfhtnode, const void* key0)
{
  const FibEntry* entry = container_of(lfhtnode, FibEntry, lfhtnode);
  const LName* key = (const LName*)key0;
  return entry->nameL == key->length && memcmp(entry->nameV, key->value, key->length) == 0;
}

bool
Fib_AllocBulk(Fib* fib, FibEntry* entries[], unsigned count)
{
  int res = rte_mempool_get_bulk(fib->mp, (void**)entries, count);
  if (unlikely(res != 0)) {
    return false;
  }

  for (unsigned i = 0; i < count; ++i) {
    FibEntry* entry = entries[i];
    *entry = (const FibEntry){ 0 };
    cds_lfht_node_init(&entry->lfhtnode);
  }
  return true;
}

void
Fib_Free(Fib* fib, FibEntry* entry)
{
  rte_mempool_put(fib->mp, entry);
}

void
Fib_Clear(Fib* fib)
{
  rcu_read_lock();
  struct cds_lfht_iter it;
  struct cds_lfht_node* node;
  cds_lfht_for_each(fib->lfht, &it, node)
  {
    FibEntry* oldEntry = container_of(node, FibEntry, lfhtnode);
    FibEntry* oldReal = FibEntry_GetReal(oldEntry);
    if (likely(oldReal != NULL)) {
      StrategyCode_Unref(oldReal->strategy);
    }
    cds_lfht_del(fib->lfht, node);
  }
  rcu_read_unlock();
}

__attribute__((nonnull)) static void
Fib_Free_(FibEntry* entry)
{
  rte_mempool_put(rte_mempool_from_obj(entry), entry);
}

__attribute__((nonnull)) static void
Fib_RcuFreeVirt_(struct rcu_head* rcuhead)
{
  FibEntry* oldVirt = container_of(rcuhead, FibEntry, rcuhead);
  NDNDPDK_ASSERT(oldVirt->maxDepth > 0);
  NDNDPDK_ASSERT(oldVirt->nNexthops == 0);
  Fib_Free_(oldVirt);
}

__attribute__((nonnull)) static void
Fib_RcuFreeReal_(struct rcu_head* rcuhead)
{
  FibEntry* oldReal = container_of(rcuhead, FibEntry, rcuhead);
  NDNDPDK_ASSERT(oldReal->maxDepth == 0);
  NDNDPDK_ASSERT(oldReal->nNexthops > 0);
  Fib_Free_(oldReal);
}

static void
Fib_FreeOld_(FibEntry* entry, Fib_FreeOld freeVirt, Fib_FreeOld freeReal)
{
  FibEntry* oldReal = NULL;
  if (entry != NULL && entry->maxDepth > 0) {
    FibEntry* oldVirt = entry;
    oldReal = oldVirt->realEntry;
    NDNDPDK_ASSERT(freeVirt != Fib_FreeOld_MustNotExist);
    if (freeVirt == Fib_FreeOld_Yes || freeVirt == Fib_FreeOld_YesIfExists) {
      call_rcu(&oldVirt->rcuhead, Fib_RcuFreeVirt_);
    }
  } else {
    oldReal = entry;
    NDNDPDK_ASSERT(freeVirt == Fib_FreeOld_MustNotExist || freeVirt == Fib_FreeOld_YesIfExists);
  }

  if (oldReal != NULL) {
    NDNDPDK_ASSERT(freeReal != Fib_FreeOld_MustNotExist);
    // reused entry is not freed but its strategy was ref'ed in Fib_Insert
    StrategyCode_Unref(oldReal->strategy);
    if (freeReal == Fib_FreeOld_Yes || freeReal == Fib_FreeOld_YesIfExists) {
      call_rcu(&oldReal->rcuhead, Fib_RcuFreeReal_);
    }
  } else {
    NDNDPDK_ASSERT(freeReal == Fib_FreeOld_MustNotExist || freeReal == Fib_FreeOld_YesIfExists);
  }
}

void
Fib_Insert(Fib* fib, FibEntry* entry, Fib_FreeOld freeVirt, Fib_FreeOld freeReal)
{
  FibEntry* newReal = entry;
  if (entry->maxDepth > 0) {
    NDNDPDK_ASSERT(entry->nNexthops == 0);
    newReal = entry->realEntry;
    entry->seqNum = ++fib->insertSeqNum;
  }
  if (newReal != NULL) {
    NDNDPDK_ASSERT(newReal->maxDepth == 0);
    NDNDPDK_ASSERT(newReal->nNexthops > 0);
    StrategyCode_Ref(newReal->strategy);
    newReal->seqNum = ++fib->insertSeqNum;
  }

  LName name = { .length = entry->nameL, .value = entry->nameV };
  uint64_t hash = LName_ComputeHash(name);
  struct cds_lfht_node* oldNode =
    cds_lfht_add_replace(fib->lfht, hash, Fib_LookupMatch_, &name, &entry->lfhtnode);
  FibEntry* oldEntry = oldNode == NULL ? NULL : container_of(oldNode, FibEntry, lfhtnode);
  Fib_FreeOld_(oldEntry, freeVirt, freeReal);
}

void
Fib_Erase(Fib* fib, FibEntry* entry, Fib_FreeOld freeVirt, Fib_FreeOld freeReal)
{
  bool ok = cds_lfht_del(fib->lfht, &entry->lfhtnode) == 0;
  NDNDPDK_ASSERT(ok);
  RTE_SET_USED(ok);
  Fib_FreeOld_(entry, freeVirt, freeReal);
}

FibEntry*
Fib_Get(Fib* fib, LName name, uint64_t hash)
{
  struct cds_lfht_iter it;
  cds_lfht_lookup(fib->lfht, hash, Fib_LookupMatch_, &name, &it);
  struct cds_lfht_node* lfhtnode = cds_lfht_iter_get_node(&it);

  static_assert(offsetof(FibEntry, lfhtnode) == 0,
                ""); // container_of(NULL, FibEntry, lfhtnode) == NULL
  return container_of(lfhtnode, FibEntry, lfhtnode);
}

__attribute__((nonnull)) static FibEntry*
Fib_GetEntryByPrefix_(Fib* fib, const PName* name, int prefixLen)
{
  uint64_t hash = PName_ComputePrefixHash(name, prefixLen);
  return Fib_Get(fib, PName_GetPrefix(name, prefixLen), hash);
}

FibEntry*
Fib_Lpm(Fib* fib, const PName* name)
{
  // first stage
  int prefixLen = name->nComps;
  if (fib->startDepth < prefixLen) {
    FibEntry* entry = Fib_GetEntryByPrefix_(fib, name, fib->startDepth);
    if (entry == NULL) { // continue to shorter prefixes
      prefixLen = fib->startDepth - 1;
    } else if (entry->maxDepth > 0) { // restart at a longest prefix
      prefixLen = fib->startDepth + entry->maxDepth;
      if (prefixLen > name->nComps) {
        prefixLen = name->nComps;
      }
    } else { // the start entry itself is a match
      return entry;
    }
  }

  // second stage
  for (; prefixLen >= 0; --prefixLen) {
    FibEntry* entry = FibEntry_GetReal(Fib_GetEntryByPrefix_(fib, name, prefixLen));
    if (entry != NULL) {
      return entry;
    }
  }

  return NULL;
}
