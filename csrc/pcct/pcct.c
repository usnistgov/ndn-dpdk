#include "pcct.h"

#include "cs.h"
#include "pit.h"

#include "../core/logger.h"
#include "../dpdk/hash.h"

INIT_ZF_LOG(Pcct);

#define uthash_malloc(sz) rte_malloc("PCCT.uthash", (sz), 0)
#define uthash_free(ptr, sz) rte_free((ptr))
#define HASH_KEYCMP(a, b, n) (!PccKey_MatchSearchKey((const PccKey*)(a), (const PccSearch*)(b)))
#define uthash_fatal(msg) rte_panic("uthash_fatal %s", msg)

#include "../vendor/uthash.h"

#undef HASH_INITIAL_NUM_BUCKETS
#undef HASH_INITIAL_NUM_BUCKETS_LOG2
#undef HASH_BKT_CAPACITY_THRESH
#undef HASH_EXPAND_BUCKETS
#define HASH_INITIAL_NUM_BUCKETS (pcct->nKeyHtBuckets)
#define HASH_INITIAL_NUM_BUCKETS_LOG2 (rte_log2_u32(HASH_INITIAL_NUM_BUCKETS))
#define HASH_BKT_CAPACITY_THRESH UINT_MAX
#define HASH_EXPAND_BUCKETS(hh, tbl, oomed) Pcct_KeyHt_Expand_(tbl)

#define PCCT_TOKEN_MASK (((uint64_t)1 << 48) - 1)

__attribute__((nonnull)) static void
Pcct_KeyHt_Expand_(UT_hash_table* tbl)
{
  ZF_LOGE("KeyHt(%p) Expand-rejected num_items=%u num_buckets=%u", tbl, tbl->num_items,
          tbl->num_buckets);
}

bool
Pcct_Init(Pcct* pcct, const char* id, uint32_t maxEntries, unsigned numaSocket)
{
  pcct->nKeyHtBuckets = rte_align32prevpow2(maxEntries);
  pcct->lastToken = PCCT_TOKEN_MASK - 16;

  struct rte_hash_parameters tokenHtParams = {
    .name = id,
    .entries = 2 * maxEntries,   // keep occupancy under 50%
    .key_len = sizeof(uint64_t), // 64-bit compares faster than 48-bit
    .hash_func = Hash_Hash64,
    .socket_id = numaSocket,
  };
  pcct->tokenHt = rte_hash_create(&tokenHtParams);
  if (unlikely(pcct->tokenHt == NULL)) {
    return false;
  }
  rte_hash_set_cmp_func(pcct->tokenHt, Hash_Equal64);

  ZF_LOGI("%p Init()", pcct);
  return true;
}

void
Pcct_Clear(Pcct* pcct)
{
  ZF_LOGI("%p Close()", pcct);
  if (pcct->tokenHt != NULL) {
    rte_hash_free(pcct->tokenHt);
  }
  HASH_CLEAR(hh, pcct->keyHt);
}

PccEntry*
Pcct_Insert(Pcct* pcct, PccSearch* search, bool* isNew)
{
  uint64_t hash = PccSearch_ComputeHash(search);

  PccEntry* entry = NULL;
  HASH_FIND_BYHASHVALUE(hh, pcct->keyHt, search, 0, hash, entry);
  if (entry != NULL) {
    *isNew = false;
    return entry;
  }

  void* objs[1 + PCC_KEY_MAX_EXTS];
  int nExts = PccKey_CountExtensions(search);
  int res = rte_mempool_get_bulk(pcct->mp, objs, 1 + nExts);
  if (unlikely(res != 0)) {
    ZF_LOGE("%p Insert() table-full", pcct);
    return NULL;
  }
  entry = (PccEntry*)objs[0];

  PccKey_CopyFromSearch(&entry->key, search, (PccKeyExt**)&objs[1], nExts);
  entry->tokenQword = 0;
  entry->slot1.pccEntry = NULL;
  entry->ext = NULL;
  HASH_ADD_BYHASHVALUE(hh, pcct->keyHt, key, 0, hash, entry);
  *isNew = true;

  char debugStringBuffer[PccSearchDebugStringLength];
  ZF_LOGD("%p Insert(%016" PRIx64 ", %s) %p", pcct, hash,
          PccSearch_ToDebugString(search, debugStringBuffer), entry);
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
  NDNDPDK_ASSERT(!entry->hasToken);

  uint64_t token = pcct->lastToken;
  while (true) {
    ++token;
    if (unlikely(token > PCCT_TOKEN_MASK)) {
      token = 1;
    }

    hash_sig_t hash = rte_hash_hash(pcct->tokenHt, &token);
    if (unlikely(rte_hash_lookup_with_hash(pcct->tokenHt, &token, hash) >= 0)) {
      // token is in use
      continue;
    }

    int res = rte_hash_add_key_with_hash_data(pcct->tokenHt, &token, hash, entry);
    if (likely(res == 0)) {
      break;
    }
    // token insertion failed
    NDNDPDK_ASSERT(res == -ENOSPC);
  }
  pcct->lastToken = token;

  entry->token = token;
  entry->hasToken = true;
  ZF_LOGD("%p AddToken(%p) %012" PRIx64, pcct, entry, token);
  return token;
}

void
Pcct_RemoveToken_(Pcct* pcct, PccEntry* entry)
{
  NDNDPDK_ASSERT(entry->hasToken);
  NDNDPDK_ASSERT(Pcct_FindByToken(pcct, entry->token) == entry);

  uint64_t token = entry->token;
  ZF_LOGD("%p RemoveToken(%p, %012" PRIx64 ")", pcct, entry, token);

  entry->hasToken = false;
  int res = rte_hash_del_key(pcct->tokenHt, &token);
  NDNDPDK_ASSERT(res >= 0);
}

PccEntry*
Pcct_FindByToken(const Pcct* pcct, uint64_t token)
{
  token &= PCCT_TOKEN_MASK;

  void* entry = NULL;
  int res = rte_hash_lookup_data(pcct->tokenHt, &token, &entry);
  NDNDPDK_ASSERT((res >= 0 && entry != NULL) || (res == -ENOENT && entry == NULL));
  return (PccEntry*)entry;
}

void
PcctEraseBatch_EraseBurst_(PcctEraseBatch* peb)
{
  NDNDPDK_ASSERT(peb->pcct != NULL);
  int nObjs = peb->nEntries;
  for (int i = 0; i < peb->nEntries; ++i) {
    PccEntry* entry = (PccEntry*)peb->objs[i];
    ZF_LOGD("%p Erase(%p)", peb->pcct, entry);
    NDNDPDK_ASSERT(!entry->hasEntries);
    Pcct_RemoveToken(peb->pcct, entry);
    HASH_DELETE(hh, peb->pcct->keyHt, entry);

    nObjs += PccKey_StripExts(&entry->key, (PccKeyExt**)&peb->objs[nObjs]);
    if (entry->ext != NULL) {
      peb->objs[nObjs++] = entry->ext;
    }
    NDNDPDK_ASSERT((size_t)nObjs < RTE_DIM(peb->objs));
  }
  rte_mempool_put_bulk(peb->pcct->mp, peb->objs, nObjs);
  peb->nEntries = 0;
}
