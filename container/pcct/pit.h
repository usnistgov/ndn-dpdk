#ifndef NDN_DPDK_CONTAINER_PCCT_PIT_H
#define NDN_DPDK_CONTAINER_PCCT_PIT_H

/// \file

#include "pcct.h"

/** \brief The Pending Interest Table (PIT).
 *
 *  Pit* is Pcct*.
 */
typedef struct Pit
{
} Pit;

static Pcct*
Pcct_FromPit(const Pit* pit)
{
  return (Pcct*)pit;
}

static Pit*
Pcct_GetPit(const Pcct* pcct)
{
  return (Pit*)pcct;
}

static PitPriv*
Pit_GetPriv(const Pit* pit)
{
  return &Pcct_GetPriv(Pcct_FromPit(pit))->pitPriv;
}

/** \brief Get number of PIT entries.
 */
static uint32_t
Pit_CountEntries(const Pit* pit)
{
  return Pit_GetPriv(pit)->nEntries;
}

/** \brief Result of PIT insertion.
 */
typedef PccEntry* PitInsertResult;

typedef enum PitInsertResultKind {
  PIT_INSERT_FULL, ///< PIT is full, cannot insert
  PIT_INSERT_PIT,  ///< created or found PIT entry
  PIT_INSERT_CS,   ///< found existing CS entry, cannot insert PIT entry
} PitInsertResultKind;

static PitInsertResultKind
PitInsertResult_GetKind(PitInsertResult res)
{
  if (unlikely(res == NULL)) {
    return PIT_INSERT_FULL;
  }
  if (res->hasCsEntry) {
    return PIT_INSERT_CS;
  }
  assert(res->hasPitEntry);
  return PIT_INSERT_PIT;
}

static PitEntry*
PitInsertResult_GetPitEntry(PitInsertResult res)
{
  assert(PitInsertResult_GetKind(res) == PIT_INSERT_PIT);
  return &res->pitEntry;
}

static CsEntry*
PitInsertResult_GetCsEntry(PitInsertResult res)
{
  assert(PitInsertResult_GetKind(res) == PIT_INSERT_CS);
  return &res->csEntry;
}

/** \brief Insert or find a PIT entry for the given Interest.
 */
PitInsertResult Pit_Insert(Pit* pit, Packet* npkt);

/** \brief Assign a token to a PIT entry.
 *  \return New or existing token.
 */
static uint64_t
Pit_AddToken(Pit* pit, PitEntry* entry)
{
  return Pcct_AddToken(Pcct_FromPit(pit), PccEntry_FromPitEntry(entry));
}

/** \brief Erase a PIT entry but retain the PccEntry.
 *  \return enclosing PccEntry.
 *  \post \p entry is no longer valid.
 */
PccEntry* __Pit_RawErase(Pit* pit, PitEntry* entry);

/** \brief Erase a PIT entry.
 *  \post \p entry is no longer valid.
 */
static void
Pit_Erase(Pit* pit, PitEntry* entry)
{
  PccEntry* pccEntry = __Pit_RawErase(pit, entry);
  Pcct_Erase(Pcct_FromPit(pit), pccEntry);
}

/** \brief Find a PIT entry for the given token.
 *  \param token the token, only lower 48 bits are significant.
 */
static PitEntry*
Pit_Find(Pit* pit, uint64_t token)
{
  PccEntry* pccEntry = Pcct_FindByToken(Pcct_FromPit(pit), token);
  if (likely(pccEntry != NULL && pccEntry->hasPitEntry)) {
    return PccEntry_GetPitEntry(pccEntry);
  }
  return NULL;
}

#endif // NDN_DPDK_CONTAINER_PCCT_PIT_H
