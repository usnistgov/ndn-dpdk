#ifndef NDN_DPDK_CONTAINER_PCCT_CS_ENTRY_H
#define NDN_DPDK_CONTAINER_PCCT_CS_ENTRY_H

/// \file

#include "common.h"

/** \brief A CS entry.
 *
 *  This struct is enclosed in \p PccEntry.
 */
typedef struct CsEntry
{
  Packet* data; ///< the Data packet
} CsEntry;

#endif // NDN_DPDK_CONTAINER_PCCT_CS_ENTRY_H
