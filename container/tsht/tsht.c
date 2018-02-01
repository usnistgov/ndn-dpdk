#include "tsht.h"

#include <rte_mempool.h>

// lfht must be capable of storing the full hash value.
static_assert(sizeof(((struct cds_lfht_node*)NULL)->reverse_hash) ==
                sizeof(uint64_t),
              "");

Tsht*
Tsht_New(const char* id, uint32_t maxEntries, uint32_t nBuckets,
         Tsht_Match match, size_t sizeofEntry, size_t sizeofHead,
         unsigned numaSocket)
{
  uint32_t nodeSize = sizeof(TshtNode) + sizeofEntry;
  uint32_t privSize = sizeof(TshtPriv) + sizeofHead;
  Tsht* ht =
    rte_mempool_create(id, maxEntries, nodeSize, 0, privSize, NULL, NULL, NULL,
                       NULL, numaSocket, MEMPOOL_F_SP_PUT | MEMPOOL_F_SC_GET);
  if (unlikely(ht == NULL)) {
    return NULL;
  }

  TshtPriv* htp = Tsht_GetPriv(ht);
  htp->match = (cds_lfht_match_fct)match;
  htp->lfht = cds_lfht_new(nBuckets, nBuckets, nBuckets, 0, NULL);
  if (unlikely(htp->lfht == NULL)) {
    rte_mempool_free(ht);
    return NULL;
  }

  return ht;
}

void
Tsht_Close(Tsht* ht)
{
  assert(false); // not implemented
}

TshtEntryPtr
Tsht_Alloc(Tsht* ht)
{
  void* node0 = NULL;
  int res = rte_mempool_get(ht, &node0);
  if (unlikely(res != 0)) {
    return NULL;
  }

  TshtNode* node = (TshtNode*)node0;
  cds_lfht_node_init(&node->lfhtnode);
  return node->entry;
}

static void
Tsht_FreeNode(struct rcu_head* rcuhead)
{
  TshtNode* node = caa_container_of(rcuhead, TshtNode, rcuhead);
  Tsht* ht = rte_mempool_from_obj(node);
  rte_mempool_put(ht, node);
}

bool
Tsht_Insert(Tsht* ht, uint64_t hash, const void* key, TshtEntryPtr newEntry)
{
  TshtPriv* htp = Tsht_GetPriv(ht);
  TshtNode* newNode = TshtNode_FromEntry(newEntry);

  struct cds_lfht_node* oldLfhtNode =
    cds_lfht_add_replace(htp->lfht, hash, htp->match, key, &newNode->lfhtnode);

  if (oldLfhtNode == NULL) {
    return true;
  }

  TshtNode* oldNode = caa_container_of(oldLfhtNode, TshtNode, lfhtnode);
  call_rcu(&oldNode->rcuhead, Tsht_FreeNode);
  return false;
}

bool
Tsht_Erase(Tsht* ht, TshtEntryPtr entry)
{
  TshtPriv* htp = Tsht_GetPriv(ht);
  TshtNode* node = TshtNode_FromEntry(entry);
  bool ok = cds_lfht_del(htp->lfht, &node->lfhtnode) == 0;

  if (likely(ok)) {
    call_rcu(&node->rcuhead, Tsht_FreeNode);
  }
  return ok;
}

TshtEntryPtr
Tsht_Find(Tsht* ht, uint64_t hash, const void* key)
{
  TshtPriv* htp = Tsht_GetPriv(ht);

  struct cds_lfht_iter it;
  cds_lfht_lookup(htp->lfht, hash, htp->match, key, &it);
  struct cds_lfht_node* lfhtnode = cds_lfht_iter_get_node(&it);

  if (lfhtnode == NULL) {
    return NULL;
  }
  return caa_container_of(lfhtnode, TshtNode, lfhtnode)->entry;
}
