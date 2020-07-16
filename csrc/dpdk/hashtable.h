#ifndef NDNDPDK_DPDK_HASH_H
#define NDNDPDK_DPDK_HASH_H

/** @file */

#include "../core/common.h"
#include <rte_hash.h>

/**
 * @brief Create a hashtable.
 * @param params rte_hash parameters.
 *               Required: @c name , @c entries , @c key_len , @c socket_id .
 *               Optional: @c hash_func , @c extra_flag .
 * @return the hashtable.
 * @retval NULL error. Error code is in @c rte_errno .
 */
struct rte_hash*
HashTable_New(struct rte_hash_parameters params);

#endif // NDNDPDK_DPDK_HASH_H
