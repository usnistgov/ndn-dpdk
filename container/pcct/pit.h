#ifndef NDN_DPDK_CONTAINER_PCCT_PIT_H
#define NDN_DPDK_CONTAINER_PCCT_PIT_H

/// \file

#include "pcct.h"
#include "pit-result.h"

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

/** \brief Insert or find a PIT entry for the given Interest.
 *  \param npkt Interest packet. PIT references it if creating a new PIT entry;
 *              caller may use it until \p PitEntry_DnRxInterest.
 *
 *  If there is a CS match, return the CS entry. If there is a PIT match,
 *  return the PIT entry. Otherwise, unless the PCCT is full, insert and
 *  initialize a PIT entry.
 */
PitResult Pit_Insert(Pit* pit, Packet* npkt);

/** \brief Get a token of a PIT entry.
 */
static uint64_t
Pit_GetEntryToken(Pit* pit, PitEntry* entry)
{
  PccEntry* pccEntry = PccEntry_FromPitEntry(entry);
  assert(pccEntry->hasToken);
  return pccEntry->token;
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

/** \brief Find PIT entries matching a Data.
 *  \param npkt Data packet, its token will be used.
 */
PitResult Pit_FindByData(Pit* pit, Packet* npkt);

/** \brief Find PIT entry matching a Nack.
 *  \param npkt Nack packet, its token will be used.
 */
PitEntry* Pit_FindByNack(Pit* pit, Packet* npkt);

#endif // NDN_DPDK_CONTAINER_PCCT_PIT_H
