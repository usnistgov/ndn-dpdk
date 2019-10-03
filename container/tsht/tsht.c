#include "tsht.h"

// lfht must be capable of storing the full hash value.
static_assert(sizeof(((struct cds_lfht_node*)NULL)->reverse_hash) ==
                sizeof(uint64_t),
              "");

Tsht*
Tsht_New(const char* id,
         uint32_t maxEntries,
         uint32_t nBuckets,
         Tsht_Match match,
         Tsht_Finalize finalize,
         size_t sizeofEntry,
         size_t sizeofHead,
         unsigned numaSocket)
{
  uint32_t nodeSize = sizeof(TshtNode) + sizeofEntry;
  uint32_t privSize = sizeof(TshtPriv) + sizeofHead;
  Tsht* ht = (Tsht*)rte_mempool_create(id,
                                       maxEntries,
                                       nodeSize,
                                       0,
                                       privSize,
                                       NULL,
                                       NULL,
                                       NULL,
                                       NULL,
                                       numaSocket,
                                       MEMPOOL_F_SP_PUT | MEMPOOL_F_SC_GET);
  if (unlikely(ht == NULL)) {
    return NULL;
  }

  TshtPriv* htp = Tsht_GetPriv(ht);
  htp->match = (cds_lfht_match_fct)match;
  htp->finalize = finalize;
  htp->lfht = cds_lfht_new(nBuckets, nBuckets, nBuckets, 0, NULL);
  if (unlikely(htp->lfht == NULL)) {
    rte_mempool_free(Tsht_ToMempool(ht));
    return NULL;
  }

  return ht;
}

void
Tsht_Close(Tsht* ht)
{
  TshtPriv* htp = Tsht_GetPriv(ht);

  rcu_read_lock();
  struct cds_lfht_iter it;
  struct cds_lfht_node* node;
  cds_lfht_for_each(htp->lfht, &it, node) { cds_lfht_del(htp->lfht, node); }
  rcu_read_unlock();

  int res = cds_lfht_destroy(htp->lfht, NULL);
  assert(res == 0);
  rte_mempool_free(Tsht_ToMempool(ht));
}

bool
Tsht_AllocBulk(Tsht* ht, TshtNode* nodes[], unsigned count)
{
  int res = rte_mempool_get_bulk(Tsht_ToMempool(ht), (void**)nodes, count);
  if (unlikely(res != 0)) {
    return false;
  }

  for (unsigned i = 0; i < count; ++i) {
    cds_lfht_node_init(&nodes[i]->lfhtnode);
  }
  return true;
}

void
Tsht_Free(Tsht* ht, TshtNode* node)
{
  rte_mempool_put(Tsht_ToMempool(ht), node);
}

static void
Tsht_FreeNode(struct rcu_head* rcuhead)
{
  TshtNode* node = container_of(rcuhead, TshtNode, rcuhead);
  struct rte_mempool* mempool = rte_mempool_from_obj(node);
  Tsht* ht = (Tsht*)mempool;
  TshtPriv* htp = Tsht_GetPriv(ht);
  (*htp->finalize)(node, ht);
  rte_mempool_put(mempool, node);
}

bool
Tsht_Insert(Tsht* ht, uint64_t hash, const void* key, TshtNode* newNode)
{
  TshtPriv* htp = Tsht_GetPriv(ht);
  struct cds_lfht_node* oldLfhtNode =
    cds_lfht_add_replace(htp->lfht, hash, htp->match, key, &newNode->lfhtnode);

  if (oldLfhtNode == NULL) {
    return true;
  }

  TshtNode* oldNode = container_of(oldLfhtNode, TshtNode, lfhtnode);
  call_rcu(&oldNode->rcuhead, Tsht_FreeNode);
  return false;
}

bool
Tsht_Erase(Tsht* ht, TshtNode* node)
{
  TshtPriv* htp = Tsht_GetPriv(ht);
  bool ok = cds_lfht_del(htp->lfht, &node->lfhtnode) == 0;

  if (likely(ok)) {
    call_rcu(&node->rcuhead, Tsht_FreeNode);
  }
  return ok;
}

TshtNode*
Tsht_Find(Tsht* ht, uint64_t hash, const void* key)
{
  TshtPriv* htp = Tsht_GetPriv(ht);

  struct cds_lfht_iter it;
  cds_lfht_lookup(htp->lfht, hash, htp->match, key, &it);
  struct cds_lfht_node* lfhtnode = cds_lfht_iter_get_node(&it);

  static_assert(offsetof(TshtNode, lfhtnode) == 0,
                ""); // container_of(NULL, TshtNode, lfhtnode) == NULL
  return container_of(lfhtnode, TshtNode, lfhtnode);
}
