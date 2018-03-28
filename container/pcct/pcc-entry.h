#ifndef NDN_DPDK_CONTAINER_PCCT_PCC_ENTRY_H
#define NDN_DPDK_CONTAINER_PCCT_PCC_ENTRY_H

/// \file

#include "cs-entry.h"
#include "pcc-key.h"
#include "pit-entry.h"
#include "../../core/uthash.h"

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
      bool hasPitEntry0 : 1;
      bool hasPitEntry1 : 1;
      bool hasCsEntry : 1;
      int : 12;
      uint64_t token : 48;
    } __rte_packed;
    struct
    {
      int : 1;
      int hasEntries : 3;
      uint64_t : 56;
    } __rte_packed;
    struct
    {
      int : 1;
      int hasPitEntries : 2;
      uint64_t : 57;
    } __rte_packed;
    uint64_t __tokenQword;
  };

  union
  {
    PitEntry pitEntry0; ///< PIT entry of MustBeFresh=0
    CsEntry csEntry;    ///< CS entry
  };
  PitEntry pitEntry1; ///< PIT entry of MustBeFresh=1
} PccEntry;

/** \brief Get PIT entry of MustBeFresh=0 from \p entry.
 */
static PitEntry*
PccEntry_GetPitEntry0(PccEntry* entry)
{
  assert(entry->hasPitEntry0);
  return &entry->pitEntry0;
}

/** \brief Get PIT entry of MustBeFresh=1 from \p entry.
 */
static PitEntry*
PccEntry_GetPitEntry1(PccEntry* entry)
{
  assert(entry->hasPitEntry1);
  return &entry->pitEntry1;
}

/** \brief Access \c PccEntry struct containing given PIT entry of MustBeFresh=0.
 */
static PccEntry*
PccEntry_FromPitEntry0(PitEntry* pitEntry)
{
  assert(pitEntry->mustBeFresh == false);
  PccEntry* entry = container_of(pitEntry, PccEntry, pitEntry0);
  assert(entry->hasPitEntry0);
  return entry;
}

/** \brief Access \c PccEntry struct containing given PIT entry of MustBeFresh=1.
 */
static PccEntry*
PccEntry_FromPitEntry1(PitEntry* pitEntry)
{
  assert(pitEntry->mustBeFresh == true);
  PccEntry* entry = container_of(pitEntry, PccEntry, pitEntry1);
  assert(entry->hasPitEntry1);
  return entry;
}

/** \brief Access \c PccEntry struct containing given PIT entry.
 */
static PccEntry*
PccEntry_FromPitEntry(PitEntry* pitEntry)
{
  if (pitEntry->mustBeFresh) {
    return PccEntry_FromPitEntry1(pitEntry);
  }
  return PccEntry_FromPitEntry0(pitEntry);
}

/** \brief Get CS entry from \p entry.
 */
static CsEntry*
PccEntry_GetCsEntry(PccEntry* entry)
{
  assert(entry->hasCsEntry);
  return &entry->csEntry;
}

/** \brief Access \c PccEntry struct containing given CS entry.
 */
static PccEntry*
PccEntry_FromCsEntry(CsEntry* csEntry)
{
  PccEntry* entry = container_of(csEntry, PccEntry, csEntry);
  assert(entry->hasCsEntry);
  return entry;
}

#endif // NDN_DPDK_CONTAINER_PCCT_PCC_ENTRY_H
