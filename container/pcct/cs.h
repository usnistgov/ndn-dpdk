#ifndef NDN_DPDK_CONTAINER_PCCT_CS_H
#define NDN_DPDK_CONTAINER_PCCT_CS_H

/// \file

#include "cs-arc.h"
#include "pcct.h"
#include "pit-result.h"

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
void
Cs_Init(Cs* cs, uint32_t capMd, uint32_t capMi);

/** \brief Get capacity in number of entries.
 */
uint32_t
Cs_GetCapacity(const Cs* cs, CsListId cslId);

/** \brief Get number of entries.
 */
uint32_t
Cs_CountEntries(const Cs* cs, CsListId cslId);

/** \brief Insert a CS entry.
 *  \param npkt the Data packet. CS takes ownership.
 *  \param pitFound result of Pit_FindByData that contains PIT entries
 *                  satisfied by this Data; its kind must not be PIT_FIND_NONE.
 *  \post PIT entries contained in \p pitFound are removed.
 */
void
Cs_Insert(Cs* cs, Packet* npkt, PitFindResult pitFound);

/** \brief Determine whether the CS entry matches an Interest during PIT insertion.
 *  \param pccEntry the PCC entry containing CS entry
 *  \post the CS entry is erased if it would conflict with a PIT entry for the Interest.
 */
bool
Cs_MatchInterest_(Cs* cs, PccEntry* pccEntry, Packet* interestNpkt);

/** \brief Erase a CS entry.
 *  \post \p entry is no longer valid.
 */
void
Cs_Erase(Cs* cs, CsEntry* entry);

#endif // NDN_DPDK_CONTAINER_PCCT_CS_H
