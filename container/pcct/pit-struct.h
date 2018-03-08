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
  uint64_t nEntries;
  MinSched* timeoutSched;

  struct rte_mempool* headerMp;   ///< mempool for Interest header
  struct rte_mempool* guiderMp;   ///< mempool for Interest guiders
  struct rte_mempool* indirectMp; ///< mempool for indirect mbufs
} PitPriv;

#endif // NDN_DPDK_CONTAINER_PCCT_PIT_STRUCT_H
