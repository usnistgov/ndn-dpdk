#include "pcct.h"

#include "cs.h"
#include "pit.h"

#include "../../core/logger.h"

#include <rte_jhash.h>

INIT_ZF_LOG(Pcct);

#undef uthash_malloc
#undef uthash_free
#undef uthash_memcmp
#define uthash_malloc(sz) rte_malloc("PCCT.uthash", (sz), 0)
#define uthash_free(ptr, sz) rte_free((ptr))
#define uthash_memcmp(a, b, n)                                                 \
  (!PccKey_MatchSearchKey((const PccKey*)(a), (const PccSearch*)(b)))

#define PCCT_TOKEN_MASK (((uint64_t)1 << 48) - 1)

static uint32_t
Pcct_TokenHt_Hash_(const void* key, uint32_t keyLen, uint32_t initVal)
{
  assert(keyLen == sizeof(uint32_t) * 2);
  const uint32_t* words = (const uint32_t*)key;
  return rte_jhash_2words(words[0], words[1], initVal);
}

static int
Pcct_TokenHt_Cmp_(const void* key1, const void* key2, size_t kenLen)
{
  assert(kenLen == sizeof(uint64_t));
  return *(const uint64_t*)key1 != *(const uint64_t*)key2;
}

Pcct*
Pcct_New(const char* id, uint32_t maxEntries, unsigned numaSocket)
{
  char tokenHtName[RTE_HASH_NAMESIZE];
  int tokenHtNameLen =
    snprintf(tokenHtName, sizeof(tokenHtName), "%s.token", id);
  if (tokenHtNameLen < 0 || tokenHtNameLen >= sizeof(tokenHtName)) {
    rte_errno = ENAMETOOLONG;
    return NULL;
  }

  Pcct* pcct =
    (Pcct*)rte_mempool_create(id,
                              maxEntries,
                              RTE_MAX(sizeof(PccEntry), sizeof(PccEntryExt)),
                              0,
                              sizeof(PcctPriv),
                              NULL,
                              NULL,
                              NULL,
                              NULL,
                              numaSocket,
                              MEMPOOL_F_SP_PUT | MEMPOOL_F_SC_GET);
  if (unlikely(pcct == NULL)) {
    return NULL;
  }

  PcctPriv* pcctp = Pcct_GetPriv(pcct);
  memset(pcctp, 0, sizeof(*pcctp));
  pcctp->lastToken = PCCT_TOKEN_MASK - 16;

  struct rte_hash_parameters tokenHtParams = {
    .name = tokenHtName,
    .entries = maxEntries * 2,   // keep occupancy under 50%
    .key_len = sizeof(uint64_t), // 64-bit compares faster than 48-bit
    .hash_func = Pcct_TokenHt_Hash_,
    .socket_id = numaSocket,
  };
  pcctp->tokenHt = rte_hash_create(&tokenHtParams);
  rte_hash_set_cmp_func(pcctp->tokenHt, Pcct_TokenHt_Cmp_);

  ZF_LOGI("%p New('%s')", pcct, id);
  return pcct;
}

void
Pcct_Close(Pcct* pcct)
{
  ZF_LOGI("%p Close()", pcct);

  PcctPriv* pcctp = Pcct_GetPriv(pcct);
  rte_hash_free(pcctp->tokenHt);
  HASH_CLEAR(hh, pcctp->keyHt);
  rte_mempool_free(Pcct_ToMempool(pcct));
}

PccEntry*
Pcct_Insert(Pcct* pcct, PccSearch* search, bool* isNew)
{
  PcctPriv* pcctp = Pcct_GetPriv(pcct);
  uint64_t hash = PccSearch_ComputeHash(search);

  PccEntry* entry = NULL;
  HASH_FIND_BYHASHVALUE(hh, pcctp->keyHt, search, 0, hash, entry);
  if (entry != NULL) {
    *isNew = false;
    return entry;
  }

  void* objs[1 + PCC_KEY_MAX_EXTS];
  int nExts = PccKey_CountExtensions(search);
  int res = rte_mempool_get_bulk(Pcct_ToMempool(pcct), objs, 1 + nExts);
  if (unlikely(res != 0)) {
    ZF_LOGE("%p Insert() table-full", pcct);
    return NULL;
  }
  entry = (PccEntry*)objs[0];

  PccKey_CopyFromSearch(&entry->key, search, (PccKeyExt**)&objs[1], nExts);
  entry->tokenQword = 0;
  entry->slot1.pccEntry = NULL;
  entry->ext = NULL;
  HASH_ADD_BYHASHVALUE(hh, pcctp->keyHt, key, 0, hash, entry);
  *isNew = true;

  ZF_LOGD("%p Insert(%016" PRIx64 ", %s) %p",
          pcct,
          hash,
          PccSearch_ToDebugString(search),
          entry);
  return entry;
}

void
Pcct_Erase(Pcct* pcct, PccEntry* entry)
{
  PcctEraseBatch peb = PcctEraseBatch_New(pcct);
  PcctEraseBatch_Append(&peb, entry);
  PcctEraseBatch_Finish(&peb);
}

uint64_t
Pcct_AddToken_(Pcct* pcct, PccEntry* entry)
{
  assert(!entry->hasToken);
  PcctPriv* pcctp = Pcct_GetPriv(pcct);

  uint64_t token = pcctp->lastToken;
  while (true) {
    ++token;
    if (unlikely(token > PCCT_TOKEN_MASK)) {
      token = 1;
    }

    uint32_t hash = rte_hash_hash(pcctp->tokenHt, &token);
    if (unlikely(rte_hash_lookup_with_hash(pcctp->tokenHt, &token, hash) >=
                 0)) {
      // token is in use
      continue;
    }

    int res =
      rte_hash_add_key_with_hash_data(pcctp->tokenHt, &token, hash, entry);
    if (likely(res == 0)) {
      break;
    }
    // token insertion failed
    assert(res == -ENOSPC);
  }
  pcctp->lastToken = token;

  entry->token = token;
  entry->hasToken = true;
  ZF_LOGD("%p AddToken(%p) %012" PRIx64, pcct, entry, token);
  return token;
}

void
Pcct_RemoveToken_(Pcct* pcct, PccEntry* entry)
{
  assert(entry->hasToken);
  assert(Pcct_FindByToken(pcct, entry->token) == entry);

  PcctPriv* pcctp = Pcct_GetPriv(pcct);

  uint64_t token = entry->token;
  ZF_LOGD("%p RemoveToken(%p, %012" PRIx64 ")", pcct, entry, token);

  entry->hasToken = false;
  int res __rte_unused = rte_hash_del_key(pcctp->tokenHt, &token);
  assert(res >= 0);
}

PccEntry*
Pcct_FindByToken(const Pcct* pcct, uint64_t token)
{
  PcctPriv* pcctp = Pcct_GetPriv(pcct);

  token &= PCCT_TOKEN_MASK;

  void* entry = NULL;
  int res __rte_unused = rte_hash_lookup_data(pcctp->tokenHt, &token, &entry);
  assert((res >= 0 && entry != NULL) || (res == -ENOENT && entry == NULL));
  return (PccEntry*)entry;
}

void
PcctEraseBatch_EraseBurst_(PcctEraseBatch* peb)
{
  assert(peb->pcct != NULL);
  PcctPriv* pcctp = Pcct_GetPriv(peb->pcct);
  int nObjs = peb->nEntries;
  for (int i = 0; i < peb->nEntries; ++i) {
    PccEntry* entry = (PccEntry*)peb->objs[i];
    ZF_LOGD("%p Erase(%p)", peb->pcct, entry);
    assert(!entry->hasEntries);
    Pcct_RemoveToken(peb->pcct, entry);
    HASH_DELETE(hh, pcctp->keyHt, entry);

    nObjs += PccKey_StripExts(&entry->key, (PccKeyExt**)&peb->objs[nObjs]);
    if (entry->ext != NULL) {
      peb->objs[nObjs++] = entry->ext;
    }
    assert(nObjs < RTE_DIM(peb->objs));
  }
  rte_mempool_put_bulk(Pcct_ToMempool(peb->pcct), peb->objs, nObjs);
  peb->nEntries = 0;
}
