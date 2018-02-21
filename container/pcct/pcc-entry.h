#ifndef NDN_DPDK_CONTAINER_PCCT_PCC_ENTRY_H
#define NDN_DPDK_CONTAINER_PCCT_PCC_ENTRY_H

/// \file

#include "cs-entry.h"
#include "pcc-key.h"
#include "pit-entry.h"
#include "uthash.h"

/** \brief PIT-CS composite entry.
 */
typedef struct PccEntry
{
  PccKey key;
  UT_hash_handle hh;
  union
  {
    struct
    {
      bool hasToken : 1;
      bool hasPitEntry : 1;
      bool hasCsEntry : 1;
      int : 13;
      uint64_t token : 48;
    } __rte_packed;
    uint64_t __tokenQword;
  };

  union
  {
    PitEntry pitEntry;
    CsEntry csEntry;
  };
} PccEntry;

/** \brief Get PIT entry from \p PccEntry.
 */
static PitEntry*
PccEntry_GetPitEntry(PccEntry* entry)
{
  assert(entry->hasPitEntry);
  return &entry->pitEntry;
}

/** \brief Get \p PccEntry pointer from \p PitEntry.
 */
static PccEntry*
PccEntry_FromPitEntry(PitEntry* pitEntry)
{
  PccEntry* entry =
    (PccEntry*)RTE_PTR_SUB(pitEntry, offsetof(PccEntry, pitEntry));
  assert(entry->hasPitEntry);
  return entry;
}

/** \brief Get CS entry from \p PccEntry.
 */
static CsEntry*
PccEntry_GetCsEntry(PccEntry* entry)
{
  assert(entry->hasCsEntry);
  return &entry->csEntry;
}

/** \brief Get \p PccEntry pointer from \p CsEntry.
 */
static PccEntry*
PccEntry_FromCsEntry(CsEntry* csEntry)
{
  PccEntry* entry =
    (PccEntry*)RTE_PTR_SUB(csEntry, offsetof(PccEntry, csEntry));
  assert(entry->hasCsEntry);
  return entry;
}

#endif // NDN_DPDK_CONTAINER_PCCT_PCC_ENTRY_H
