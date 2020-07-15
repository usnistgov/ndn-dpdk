#ifndef NDNDPDK_DPDK_HASH_H
#define NDNDPDK_DPDK_HASH_H

/** @file */

#include "../core/common.h"
#include <rte_hash.h>
#include <rte_jhash.h>

/** @brief Specialized @c rte_hash_function for 64-bit keys. */
__attribute__((nonnull)) static inline hash_sig_t
Hash_Hash64(const void* key, uint32_t keyLen, uint32_t initVal)
{
  const uint32_t* words = (const uint32_t*)key;
  return rte_jhash_2words(words[0], words[1], initVal);
}

/** @brief Specialized @c rte_hash_cmp_eq_t for 64-bit keys. */
__attribute__((nonnull)) static inline int
Hash_Equal64(const void* key1, const void* key2, size_t kenLen)
{
  return *(const uint64_t*)key1 != *(const uint64_t*)key2;
}

#endif // NDNDPDK_DPDK_HASH_H
