#ifndef NDN_DPDK_CONTAINER_PCCT_CS_ENTRY_H
#define NDN_DPDK_CONTAINER_PCCT_CS_ENTRY_H

/// \file

#include "cs-struct.h"

#define PIT_ENTRY_MAX_INDIRECTS 4

/** \brief A CS entry.
 *
 *  This struct is enclosed in \c PccEntry.
 */
typedef struct CsEntry
{
  CsNode node;

  Packet* data;       ///< the Data packet
  TscTime freshUntil; ///< when to become non-fresh
} CsEntry;
static_assert(offsetof(CsEntry, node) == 0, ""); // Cs.List() assumes this

/** \brief Finalize a CS entry.
 */
static void
CsEntry_Finalize(CsEntry* entry)
{
  rte_pktmbuf_free(Packet_ToMbuf(entry->data));
}

/** \brief Determine if \p entry is fresh.
 */
static bool
CsEntry_IsFresh(CsEntry* entry, TscTime now)
{
  return entry->freshUntil > now;
}

#endif // NDN_DPDK_CONTAINER_PCCT_CS_ENTRY_H
