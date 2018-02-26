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
} PitPriv;

#endif // NDN_DPDK_CONTAINER_PCCT_PIT_STRUCT_H
