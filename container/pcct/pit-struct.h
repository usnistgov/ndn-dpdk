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

  uint64_t nDataHit;  ///< how many find-by-Data found PIT entry/entries
  uint64_t nDataMiss; ///< how many find-by-Data did not find PIT entry
  uint64_t nNackHit;  ///< how many find-by-Nack found PIT entry
  uint64_t nNackMiss; ///< how many find-by-Nack did not find PIT entry

  MinSched* timeoutSched;
} PitPriv;

#endif // NDN_DPDK_CONTAINER_PCCT_PIT_STRUCT_H
