#ifndef NDN_DPDK_CONTAINER_PCCT_CS_H
#define NDN_DPDK_CONTAINER_PCCT_CS_H

/// \file

#include "pcct.h"
#include "pit-result.h"

/** \brief Bulk size of CS eviction.
 *
 *  This is also the minimum CS capacity.
 */
#define CS_EVICT_BULK 64

/** \brief Cast Pcct* as Cs*.
 */
static Cs*
Cs_FromPcct(const Pcct* pcct)
{
  return (Cs*)pcct;
}

/** \brief Cast Cs* as Pcct*.
 */
static Pcct*
Cs_ToPcct(const Cs* cs)
{
  return (Pcct*)cs;
}

/** \brief Access CsPriv* struct.
 */
static CsPriv*
Cs_GetPriv(const Cs* cs)
{
  return &Pcct_GetPriv(Cs_ToPcct(cs))->csPriv;
}

/** \brief Constructor.
 */
void Cs_Init(Cs* cs, uint32_t capacity);

/** \brief Get capacity in number of entries.
 */
static uint32_t
Cs_GetCapacity(const Cs* cs)
{
  return Cs_GetPriv(cs)->capacity;
}

/** \brief Set capacity in number of entries.
 */
void Cs_SetCapacity(Cs* cs, uint32_t capacity);

/** \brief Get number of CS entries.
 */
static uint32_t
Cs_CountEntries(const Cs* cs)
{
  return Cs_GetPriv(cs)->nEntries;
}

/** \brief Insert a CS entry.
 *  \param npkt the Data packet. CS takes ownership.
 *  \param pitFound result of Pit_FindByData that contains PIT entries
 *                  satisfied by this Data, must not be PIT_FIND_NONE.
 *  \post PIT entries contained in \p pitFound are removed.
 */
void Cs_Insert(Cs* cs, Packet* npkt, PitResult pitFound);

/** \brief Erase CS entry but retain the PccEntry.
 */
void __Cs_RawErase(Cs* cs, CsEntry* entry);

/** \brief Erase a CS entry.
 *  \post \p entry is no longer valid.
 */
void Cs_Erase(Cs* cs, CsEntry* entry);

#endif // NDN_DPDK_CONTAINER_PCCT_CS_H
