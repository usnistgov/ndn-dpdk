#include "fib.h"

__attribute__((nonnull)) static int // bool
Fib_LookupMatch_(struct cds_lfht_node* lfhtnode, const void* key0)
{
  const FibEntry* entry = container_of(lfhtnode, FibEntry, lfhtnode);
  const LName* key = (const LName*)key0;
  return entry->nameL == key->length && memcmp(entry->nameV, key->value, key->length) == 0;
}

void
Fib_Clear(Fib* fib)
{
  rcu_read_lock();
  struct cds_lfht_iter it;
  struct cds_lfht_node* node;
  cds_lfht_for_each (fib->lfht, &it, node) {
    FibEntry* oldEntry = container_of(node, FibEntry, lfhtnode);
    FibEntry* oldReal = FibEntry_GetReal(oldEntry);
    if (likely(oldReal != NULL)) {
      StrategyCode_Unref(oldReal->strategy);
    }
    cds_lfht_del(fib->lfht, node);
  }
  rcu_read_unlock();
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
    memset(entry, 0, fib->mp->elt_size);
    cds_lfht_node_init(&entry->lfhtnode);
  }
  return true;
}

void
Fib_Write(Fib* fib, FibEntry* entry)
{
  FibEntry* newReal = entry;
  if (entry->height > 0) {
    NDNDPDK_ASSERT(entry->nNexthops == 0);
    newReal = entry->realEntry;
    entry->seqNum = ++fib->insertSeqNum;
  }
  if (newReal != NULL) {
    NDNDPDK_ASSERT(newReal->height == 0);
    NDNDPDK_ASSERT(newReal->nNexthops > 0);
    StrategyCode_Ref(newReal->strategy);
    newReal->seqNum = ++fib->insertSeqNum;
  }

  LName name = { .length = entry->nameL, .value = entry->nameV };
  uint64_t hash = LName_ComputeHash(name);
  cds_lfht_add_replace(fib->lfht, hash, Fib_LookupMatch_, &name, &entry->lfhtnode);
}

void
Fib_Erase(Fib* fib, FibEntry* entry)
{
  int res = cds_lfht_del(fib->lfht, &entry->lfhtnode);
  NDNDPDK_ASSERT(res == 0);
}

__attribute__((nonnull)) static void
Fib_RcuFree_(struct rcu_head* rcuhead)
{
  FibEntry* entry = container_of(rcuhead, FibEntry, rcuhead);
  rte_mempool_put(rte_mempool_from_obj(entry), entry);
}

void
Fib_DeferredFree(Fib* fib, FibEntry* entry)
{
  call_rcu(&entry->rcuhead, Fib_RcuFree_);
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
    } else if (entry->height > 0) { // found virtual entry, restart at longest prefix
      prefixLen = fib->startDepth + entry->height;
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
