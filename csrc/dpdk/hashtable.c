#include "hashtable.h"
#include <rte_hash_crc.h>
#include <rte_random.h>

__attribute__((nonnull)) static inline hash_sig_t
HashTable_Hash64_(const void* key, uint32_t keyLen, uint32_t initVal)
{
  return rte_hash_crc_8byte(*(const uint64_t*)key, initVal);
}

__attribute__((nonnull)) static inline int
HashTable_Equal64_(const void* key1, const void* key2, size_t kenLen)
{
  return *(const uint64_t*)key1 != *(const uint64_t*)key2;
}

struct rte_hash*
HashTable_New(struct rte_hash_parameters params)
{
  if (params.hash_func == NULL) {
    if (params.key_len == 8) {
      params.hash_func = HashTable_Hash64_;
    }
    params.hash_func_init_val = rte_rand();
  }

  struct rte_hash* h = rte_hash_create(&params);
  if (h == NULL) {
    return NULL;
  }

  if (params.key_len == 8) {
    rte_hash_set_cmp_func(h, HashTable_Equal64_);
  }
  return h;
}
