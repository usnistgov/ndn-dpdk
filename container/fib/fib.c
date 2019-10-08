#include "fib.h"

static int // bool
Fib_LookupMatch(struct cds_lfht_node* lfhtnode, const void* key0)
{
  const FibEntry* entry = container_of(lfhtnode, FibEntry, lfhtnode);
  const LName* key = (const LName*)key0;
  return entry->nameL == key->length &&
         memcmp(entry->nameV, key->value, key->length) == 0;
}

static void
Fib_FreeEntry(struct rcu_head* rcuhead)
{
  FibEntry* entry = container_of(rcuhead, FibEntry, rcuhead);
  if (likely(entry->dyn != NULL) && entry->shouldFreeDyn) {
    struct rte_mempool* dynMp = rte_mempool_from_obj(entry->dyn);
    rte_mempool_put(dynMp, entry->dyn);
  }
  if (likely(entry->strategy != NULL)) {
    StrategyCode_Unref(entry->strategy);
  }
  rte_mempool_put(rte_mempool_from_obj(entry), entry);
}

Fib*
Fib_New(const char* id,
        uint32_t maxEntries,
        uint32_t nBuckets,
        unsigned numaSocket,
        uint8_t startDepth)
{
  Fib* fib = (Fib*)rte_mempool_create(id,
                                      maxEntries,
                                      sizeof(FibEntry),
                                      0,
                                      sizeof(FibPriv),
                                      NULL,
                                      NULL,
                                      NULL,
                                      NULL,
                                      numaSocket,
                                      MEMPOOL_F_SP_PUT | MEMPOOL_F_SC_GET);
  if (unlikely(fib == NULL)) {
    return NULL;
  }

  FibPriv* fibp = Fib_GetPriv(fib);
  fibp->lfht = cds_lfht_new(nBuckets, nBuckets, nBuckets, 0, NULL);
  if (unlikely(fibp->lfht == NULL)) {
    rte_mempool_free(Fib_ToMempool(fib));
    return NULL;
  }
  fibp->startDepth = startDepth;
  return fib;
}

void
Fib_Close(Fib* fib)
{
  FibPriv* fibp = Fib_GetPriv(fib);

  rcu_read_lock();
  struct cds_lfht_iter it;
  struct cds_lfht_node* node;
  cds_lfht_for_each(fibp->lfht, &it, node) { cds_lfht_del(fibp->lfht, node); }
  rcu_read_unlock();

  int res = cds_lfht_destroy(fibp->lfht, NULL);
  assert(res == 0);
  rte_mempool_free(Fib_ToMempool(fib));
}

bool
Fib_AllocBulk(Fib* fib, FibEntry* entries[], unsigned count)
{
  int res = rte_mempool_get_bulk(Fib_ToMempool(fib), (void**)entries, count);
  if (unlikely(res != 0)) {
    return false;
  }

  for (unsigned i = 0; i < count; ++i) {
    cds_lfht_node_init(&entries[i]->lfhtnode);
  }
  return true;
}

void
Fib_Free(Fib* fib, FibEntry* entry)
{
  assert(entry->strategy == NULL);
  rte_mempool_put(Fib_ToMempool(fib), entry);
}

bool
Fib_Insert(Fib* fib, FibEntry* entry)
{
  FibPriv* fibp = Fib_GetPriv(fib);

  if (likely(entry->strategy != NULL)) {
    StrategyCode_Ref(entry->strategy);
    assert(entry->dyn != NULL);
  } else {
    assert(entry->nNexthops == 0);
  }

  LName name = { .length = entry->nameL, .value = entry->nameV };
  uint64_t hash = LName_ComputeHash(name);
  struct cds_lfht_node* oldNode = cds_lfht_add_replace(
    fibp->lfht, hash, Fib_LookupMatch, &name, &entry->lfhtnode);
  if (oldNode == NULL) {
    return true;
  }

  FibEntry* oldEntry = container_of(oldNode, FibEntry, lfhtnode);
  call_rcu(&oldEntry->rcuhead, Fib_FreeEntry);
  return false;
}

void
Fib_Erase(Fib* fib, FibEntry* entry)
{
  FibPriv* fibp = Fib_GetPriv(fib);
  bool ok = cds_lfht_del(fibp->lfht, &entry->lfhtnode) == 0;

  if (likely(ok)) {
    call_rcu(&entry->rcuhead, Fib_FreeEntry);
  }
}

const FibEntry*
Fib_Find(Fib* fib, LName name, uint64_t hash)
{
  FibPriv* fibp = Fib_GetPriv(fib);

  struct cds_lfht_iter it;
  cds_lfht_lookup(fibp->lfht, hash, Fib_LookupMatch, &name, &it);
  struct cds_lfht_node* lfhtnode = cds_lfht_iter_get_node(&it);

  static_assert(offsetof(FibEntry, lfhtnode) == 0,
                ""); // container_of(NULL, FibEntry, lfhtnode) == NULL
  return container_of(lfhtnode, FibEntry, lfhtnode);
}

static const FibEntry*
Fib_GetEntryByPrefix(Fib* fib,
                     const PName* name,
                     const uint8_t* nameV,
                     uint16_t prefixLen)
{
  uint64_t hash = PName_ComputePrefixHash(name, nameV, prefixLen);
  LName key = { .length = PName_SizeofPrefix(name, nameV, prefixLen),
                .value = nameV };
  return Fib_Find(fib, key, hash);
}

const FibEntry*
Fib_Lpm_(Fib* fib, const PName* name, const uint8_t* nameV)
{
  FibPriv* fibp = Fib_GetPriv(fib);

  // first stage
  int prefixLen = name->nComps;
  if (fibp->startDepth < prefixLen) {
    const FibEntry* entry =
      Fib_GetEntryByPrefix(fib, name, nameV, fibp->startDepth);
    if (entry == NULL) { // continue to shorter prefixes
      prefixLen = fibp->startDepth - 1;
    } else if (entry->maxDepth > 0) { // restart at a longest prefix
      prefixLen = fibp->startDepth + entry->maxDepth;
      if (prefixLen > name->nComps) {
        prefixLen = name->nComps;
      }
    } else if (entry->nNexthops > 0) { // the start entry itself is a match
      return entry;
    }
  }

  // second stage
  for (; prefixLen >= 0; --prefixLen) {
    const FibEntry* entry = Fib_GetEntryByPrefix(fib, name, nameV, prefixLen);
    if (entry != NULL && entry->nNexthops > 0) {
      return entry;
    }
  }

  return NULL;
}
