#ifndef NDN_DPDK_CONTAINER_PCCT_CS_H
#define NDN_DPDK_CONTAINER_PCCT_CS_H

/// \file

#include "pcct.h"

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
void Cs_Init(Cs* cs);

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

/** \brief Insert a CS entry by replacing a PIT entry with same key.
 *  \param pitEntry the PIT entry satisfied by this Data, will be overwritten.
 *  \param npkt the Data packet. CS takes ownership of this mbuf, and may immediately
 *              free it; caller must clone the packet if it is still needed.
 */
void Cs_Insert(Cs* cs, Packet* npkt, PitEntry* pitEntry);

/** \brief Erase a CS entry.
 *  \post \p entry is no longer valid.
 */
void Cs_Erase(Cs* cs, CsEntry* entry);

#endif // NDN_DPDK_CONTAINER_PCCT_CS_H
