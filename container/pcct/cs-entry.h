#ifndef NDN_DPDK_CONTAINER_PCCT_CS_ENTRY_H
#define NDN_DPDK_CONTAINER_PCCT_CS_ENTRY_H

/// \file

#include "cs-struct.h"

/** \brief A CS entry.
 *
 *  This struct is enclosed in \p PccEntry.
 */
typedef struct CsEntry
{
  CsNode node;

  Packet* data; ///< the Data packet
} CsEntry;
static_assert(offsetof(CsEntry, node) == 0, ""); // Cs.List() assumes this

/** \brief Finalize a CS entry.
 */
static void
CsEntry_Finalize(CsEntry* entry)
{
  rte_pktmbuf_free(Packet_ToMbuf(entry->data));
}

#endif // NDN_DPDK_CONTAINER_PCCT_CS_ENTRY_H
