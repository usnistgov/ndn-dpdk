#ifndef NDN_DPDK_CONTAINER_PCCT_PIT_H
#define NDN_DPDK_CONTAINER_PCCT_PIT_H

/// \file

#include "pcct.h"

/** \brief Maximum PIT entry lifetime (millis).
 */
#define PIT_MAX_LIFETIME 120000

/** \brief Cast Pcct* as Pit*.
 */
static Pit*
Pit_FromPcct(const Pcct* pcct)
{
  return (Pit*)pcct;
}

/** \brief Cast Pit* as Pcct*.
 */
static Pcct*
Pit_ToPcct(const Pit* pit)
{
  return (Pcct*)pit;
}

/** \brief Access PitPriv* struct.
 */
static PitPriv*
Pit_GetPriv(const Pit* pit)
{
  return &Pcct_GetPriv(Pit_ToPcct(pit))->pitPriv;
}

/** \brief Constructor.
 */
void Pit_Init(Pit* pit, struct rte_mempool* headerMp,
              struct rte_mempool* guiderMp, struct rte_mempool* indirectMp);

/** \brief Get number of PIT entries.
 */
static uint32_t
Pit_CountEntries(const Pit* pit)
{
  return Pit_GetPriv(pit)->nEntries;
}

/** \brief Result of PIT insertion.
 */
typedef struct PitInsertResult
{
  uintptr_t ptr; ///< PccEntry* | PitInsertResultKind
} PitInsertResult;

typedef enum PitInsertResultKind {
  PIT_INSERT_FULL = 0, ///< PIT is full, cannot insert
  PIT_INSERT_PIT0 = 1, ///< created or found PIT entry of MustBeFresh=0
  PIT_INSERT_PIT1 = 2, ///< created or found PIT entry of MustBeFresh=1
  PIT_INSERT_CS = 3,   ///< found existing CS entry that matches the Interest

  __PIT_INSERT_MASK = 0x03,
} PitInsertResultKind;

static PitInsertResultKind
PitInsertResult_GetKind(PitInsertResult res)
{
  return (PitInsertResultKind)(res.ptr & __PIT_INSERT_MASK);
}

static PitEntry*
PitInsertResult_GetPitEntry(PitInsertResult res)
{
  PccEntry* entry = (PccEntry*)(res.ptr & ~__PIT_INSERT_MASK);
  switch (PitInsertResult_GetKind(res)) {
    case PIT_INSERT_PIT0:
      return &entry->pitEntry0;
    case PIT_INSERT_PIT1:
      return &entry->pitEntry1;
  }
  assert(false);
}

static CsEntry*
PitInsertResult_GetCsEntry(PitInsertResult res)
{
  assert(PitInsertResult_GetKind(res) == PIT_INSERT_CS);
  PccEntry* entry = (PccEntry*)(res.ptr & ~__PIT_INSERT_MASK);
  return &entry->csEntry;
}

/** \brief Insert or find a PIT entry for the given Interest.
 *  \p npkt received Interest; will not take ownership.
 *
 *  If there is a CS match, return the CS entry. If there is a PIT match,
 *  return the PIT entry. Otherwise, unless the PCCT is full, insert and
 *  initialize a PIT entry.
 */
PitInsertResult Pit_Insert(Pit* pit, Packet* npkt);

/** \brief Assign a token to a PIT entry.
 *  \return New or existing token.
 */
static uint64_t
Pit_AddToken(Pit* pit, PitEntry* entry)
{
  return Pcct_AddToken(Pit_ToPcct(pit), PccEntry_FromPitEntry(entry));
}

/** \brief Erase a PIT entry of MustBeFresh=0 but retain the PccEntry.
 *  \return enclosing PccEntry.
 *  \post \p entry is no longer valid.
 */
PccEntry* __Pit_RawErase0(Pit* pit, PitEntry* entry);

/** \brief Erase a PIT entry of MustBeFresh=1 but retain the PccEntry.
 *  \return enclosing PccEntry.
 *  \post \p entry is no longer valid.
 */
PccEntry* __Pit_RawErase1(Pit* pit, PitEntry* entry);

/** \brief Erase a PIT entry.
 *  \post \p entry is no longer valid.
 */
void Pit_Erase(Pit* pit, PitEntry* entry);

#define PIT_FIND_MAX_MATCHES 2

/** \brief Result of PIT find.
 */
typedef struct PitFindResult
{
  PitEntry* matches[PIT_FIND_MAX_MATCHES + 1];
} PitFindResult;

/** \brief Find a PIT entry for the given token.
 *  \param token the token, only lower 48 bits are significant.
 */
PitFindResult Pit_Find(Pit* pit, uint64_t token);

#endif // NDN_DPDK_CONTAINER_PCCT_PIT_H
