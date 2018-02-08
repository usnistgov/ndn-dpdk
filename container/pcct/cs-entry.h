#ifndef NDN_DPDK_CONTAINER_PCCT_CS_ENTRY_H
#define NDN_DPDK_CONTAINER_PCCT_CS_ENTRY_H

/// \file

#include "../../core/common.h"

/** \brief A CS entry.
 *
 *  This struct is enclosed in \p PcctEntry.
 */
typedef struct CsEntry
{
  struct rte_mbuf* data; ///< the Data packet
} CsEntry;

#endif // NDN_DPDK_CONTAINER_PCCT_CS_ENTRY_H
