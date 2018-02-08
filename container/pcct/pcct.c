#include "pcct.h"

#undef uthash_malloc
#undef uthash_free
#undef uthash_memcmp
#define uthash_malloc(sz) rte_malloc("PCCT.uthash", (sz), 0)
#define uthash_free(ptr, sz) rte_free((ptr))
#define uthash_memcmp(a, b, n)                                                 \
  (!PccKey_MatchSearchKey((const PccKey*)(a), (const PccSearch*)(b)))

Pcct*
Pcct_New(const char* id, uint32_t maxEntries, unsigned numaSocket)
{
  Pcct* pcct = rte_mempool_create(
    id, maxEntries, sizeof(PccEntry), 0, sizeof(PcctPriv), NULL, NULL, NULL,
    NULL, numaSocket, MEMPOOL_F_SP_PUT | MEMPOOL_F_SC_GET);
  if (unlikely(pcct == NULL)) {
    return NULL;
  }

  PcctPriv* pcctp = Pcct_GetPriv(pcct);
  memset(pcctp, 0, sizeof(*pcctp));

  return pcct;
}

void
Pcct_Close(Pcct* pcct)
{
  PcctPriv* pcctp = Pcct_GetPriv(pcct);
  HASH_CLEAR(hh, pcctp->keyHt);
  rte_mempool_free(pcct);
}

PccEntry*
Pcct_Insert(Pcct* pcct, uint64_t hash, PccSearch* search, bool* isNew)
{
  PccEntry* entry = Pcct_Find(pcct, hash, search);
  if (entry != NULL) {
    *isNew = false;
    return entry;
  }

  void* entry0;
  int res = rte_mempool_get(pcct, &entry0);
  if (unlikely(res != 0)) {
    return NULL;
  }

  entry = (PccEntry*)entry0;
  PccKey_CopyFromSearch(&entry->key, search);
  entry->__tokenQword = 0;

  PcctPriv* pcctp = Pcct_GetPriv(pcct);
  HASH_ADD_BYHASHVALUE(hh, pcctp->keyHt, key, 0, hash, entry);
  *isNew = true;
  return entry;
}

void
Pcct_Erase(Pcct* pcct, PccEntry* entry)
{
  PcctPriv* pcctp = Pcct_GetPriv(pcct);
  HASH_DELETE(hh, pcctp->keyHt, entry);
  rte_mempool_put(pcct, entry);
}

PccEntry*
Pcct_Find(const Pcct* pcct, uint64_t hash, PccSearch* search)
{
  PcctPriv* pcctp = Pcct_GetPriv(pcct);
  PccEntry* entry = NULL;
  HASH_FIND_BYHASHVALUE(hh, pcctp->keyHt, search, 0, hash, entry);
  return entry;
}

void
Pcct_AddToken(Pcct* pcct, PccEntry* entry)
{
  assert(false); // not implemented
}

void
Pcct_RemoveToken(Pcct* pcct, PccEntry* entry)
{
  assert(false); // not implemented
}

PccEntry*
Pcct_FindByToken(const Pcct* pcct, uint64_t token)
{
  assert(false); // not implemented
  return NULL;
}
