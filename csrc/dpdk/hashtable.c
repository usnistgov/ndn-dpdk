#include "hashtable.h"
#include <rte_hash_crc.h>
#include <rte_random.h>

__attribute__((nonnull)) static hash_sig_t
HashTable_Hash64(const void* key, __rte_unused uint32_t keyLen, uint32_t initVal) {
  return rte_hash_crc_8byte(*(const uint64_t*)key, initVal);
}

__attribute__((nonnull)) static int
HashTable_Equal64(const void* key1, const void* key2, __rte_unused size_t kenLen) {
  return *(const uint64_t*)key1 != *(const uint64_t*)key2;
}

struct rte_hash*
HashTable_New(struct rte_hash_parameters params) {
  if (params.hash_func == NULL) {
    if (params.key_len == 8) {
      params.hash_func = HashTable_Hash64;
    }
    params.hash_func_init_val = rte_rand();
  }

  struct rte_hash* h = rte_hash_create(&params);
  if (h == NULL) {
    return NULL;
  }

  if (params.key_len == 8) {
    rte_hash_set_cmp_func(h, HashTable_Equal64);
  }
  return h;
}
