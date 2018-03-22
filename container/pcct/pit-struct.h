#ifndef NDN_DPDK_CONTAINER_PCCT_PIT_STRUCT_H
#define NDN_DPDK_CONTAINER_PCCT_PIT_STRUCT_H

/// \file

#include "common.h"

/** \brief The Pending Interest Table (PIT).
 *
 *  Pit* is Pcct*.
 */
typedef struct Pit
{
} Pit;

/** \brief PCCT private data for PIT.
 */
typedef struct PitPriv
{
  uint64_t nEntries; ///< current number of entries

  uint64_t nInsert;   ///< how many inserts created a new PIT entry
  uint64_t nFound;    ///< how many inserts found an existing PIT entry
  uint64_t nCsMatch;  ///< how many inserts matched a CS entry
  uint64_t nAllocErr; ///< how many inserts failed due to allocation error

  uint64_t nHits;   ///< how many finds found existing PIT entries
  uint64_t nMisses; ///< how many finds did not find existing PIT entry

  MinSched* timeoutSched;

  struct rte_mempool* headerMp;   ///< mempool for Interest header
  struct rte_mempool* guiderMp;   ///< mempool for Interest guiders
  struct rte_mempool* indirectMp; ///< mempool for indirect mbufs
} PitPriv;

#endif // NDN_DPDK_CONTAINER_PCCT_PIT_STRUCT_H
