#include "hashtable.h"
#include <rte_hash_crc.h>
#include <rte_random.h>

__attribute__((nonnull)) static hash_sig_t
HashTable_Hash32(const void* key, __rte_unused uint32_t keyLen, uint32_t initVal) {
  return rte_hash_crc_4byte(*(const uint32_t*)key, initVal);
}

__attribute__((nonnull)) static int
HashTable_Equal32(const void* key1, const void* key2, __rte_unused size_t kenLen) {
  return *(const uint32_t*)key1 != *(const uint32_t*)key2;
}

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
  rte_hash_cmp_eq_t cmp = NULL;
  if (params.hash_func == NULL) {
    switch (params.key_len) {
      case 4: {
        params.hash_func = HashTable_Hash32;
        cmp = HashTable_Equal32;
        break;
      }
      case 8: {
        params.hash_func = HashTable_Hash64;
        cmp = HashTable_Equal64;
        break;
      }
    }
    params.hash_func_init_val = rte_rand();
  }

  struct rte_hash* h = rte_hash_create(&params);
  if (h == NULL) {
    return NULL;
  }

  if (cmp != NULL) {
    rte_hash_set_cmp_func(h, cmp);
  }
  return h;
}
