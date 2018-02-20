#ifndef NDN_DPDK_CONTAINER_PCCT_CS_H
#define NDN_DPDK_CONTAINER_PCCT_CS_H

/// \file

#include "pcct.h"

/** \brief The Content Store (CS).
 *
 *  Cs* is Pcct*.
 */
typedef struct Cs
{
} Cs;

static Pcct*
Pcct_FromCs(const Cs* cs)
{
  return (Pcct*)cs;
}

static Cs*
Pcct_GetCs(const Pcct* pcct)
{
  return (Cs*)pcct;
}

static CsPriv*
Cs_GetPriv(const Cs* cs)
{
  return &Pcct_GetPriv(Pcct_FromCs(cs))->csPriv;
}

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
void Cs_ReplacePitEntry(Cs* cs, PitEntry* pitEntry, struct Packet* npkt);

/** \brief Erase a CS entry.
 *  \post \p entry is no longer valid.
 */
void Cs_Erase(Cs* cs, CsEntry* entry);

#endif // NDN_DPDK_CONTAINER_PCCT_CS_H
