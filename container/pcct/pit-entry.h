#ifndef NDN_DPDK_CONTAINER_PCCT_PIT_ENTRY_H
#define NDN_DPDK_CONTAINER_PCCT_PIT_ENTRY_H

/// \file

#include "common.h"

/** \brief A PIT entry.
 *
 *  This struct is enclosed in \p PccEntry.
 */
typedef struct PitEntry
{
  bool mustBeFresh;
} PitEntry;

#endif // NDN_DPDK_CONTAINER_PCCT_PIT_ENTRY_H
