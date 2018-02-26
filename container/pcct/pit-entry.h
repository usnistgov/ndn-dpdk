#ifndef NDN_DPDK_CONTAINER_PCCT_PIT_ENTRY_H
#define NDN_DPDK_CONTAINER_PCCT_PIT_ENTRY_H

/// \file

#include "pit-dn.h"
#include "pit-struct.h"
#include "pit-up.h"

#define PIT_ENTRY_MAX_DNS 8
#define PIT_ENTRY_MAX_UPS 8

/** \brief A PIT entry.
 *
 *  This struct is enclosed in \p PccEntry.
 */
typedef struct PitEntry
{
  Packet* npkt; ///< representative Interest packet

  MinTmr timeout;         ///< timeout timer
  uint64_t lastRxTime;    ///< last RX time (TSC)
  uint64_t suppressUntil; ///< suppression ending time (TSC)

  bool canBePrefix; ///< any RX Interest has CanBePrefix?
  bool mustBeFresh; ///< entry for MustBeFresh=0 or MustBeFresh=1?

  PitDn dns[PIT_ENTRY_MAX_DNS];
  PitUp ups[PIT_ENTRY_MAX_UPS];
} PitEntry;

/** \brief Initialize a PIT entry.
 */
static void
PitEntry_Init(Pit* pit, PitEntry* entry, Packet* npkt)
{
  PInterest* interest = Packet_GetInterestHdr(npkt);
  entry->npkt = npkt;
  MinTmr_Init(&entry->timeout);
  entry->canBePrefix = interest->canBePrefix;
  entry->mustBeFresh = interest->mustBeFresh;
  entry->dns[0].face = FACEID_INVALID;
  entry->ups[0].face = FACEID_INVALID;
}

/** \brief Refresh downstream record for RX Interest.
 *  \param entry PIT entry, must be initialized.
 *  \param npkt received Interest packet.
 *  \return index of downstream record, or -1 if no slot is available.
 */
int PitEntry_DnRxInterest(Pit* pit, PitEntry* entry, Packet* npkt);

/** \brief Prepare TX Data according to downstream record.
 *  \param entry PIT entry.
 */
int PitEntry_DnTxData(Pit* pit, PitEntry* entry, FaceId face, Packet* npkt);

#endif // NDN_DPDK_CONTAINER_PCCT_PIT_ENTRY_H
