#ifndef NDN_DPDK_CONTAINER_PCCT_PIT_H
#define NDN_DPDK_CONTAINER_PCCT_PIT_H

/// \file

#include "../../ndn/interest-pkt.h"
#include "pcct.h"

/** \brief The Pending Interest Table (PIT).
 */
typedef Pcct Pit;

static inline Pcct*
Pcct_FromPit(const Pit* pit)
{
  return (Pcct*)pit;
}

static inline Pit*
Pcct_GetPit(const Pcct* pcct)
{
  return (Pit*)pcct;
}

#define Pit_GetPriv(pit) (&Pcct_GetPriv((pit))->pitPriv)

/** \brief Get number of PIT entries.
 */
static inline uint32_t
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

static inline PitInsertResultKind
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

static inline PitEntry*
PitInsertResult_GetPitEntry(PitInsertResult res)
{
  assert(PitInsertResult_GetKind(res) == PIT_INSERT_PIT);
  return &res->pitEntry;
}

static inline CsEntry*
PitInsertResult_GetCsEntry(PitInsertResult res)
{
  assert(PitInsertResult_GetKind(res) == PIT_INSERT_CS);
  return &res->csEntry;
}

/** \brief Insert or find a PIT entry for the given Interest.
 */
PitInsertResult Pit_Insert(Pit* pit, const InterestPkt* interest);

/** \brief Erase a PIT entry but retain the PccEntry.
 *  \return enclosing PccEntry.
 *  \post \p entry is no longer valid.
 */
PccEntry* __Pit_RawErase(Pit* pit, PitEntry* entry);

/** \brief Erase a PIT entry.
 *  \post \p entry is no longer valid.
 */
void Pit_Erase(Pit* pit, PitEntry* entry);

/** \brief Find a PIT entry for the given token.
 */
PitEntry* Pit_Find(Pit* pit, uint64_t token);

#endif // NDN_DPDK_CONTAINER_PCCT_PIT_H
