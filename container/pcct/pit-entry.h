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
 *  This struct is enclosed in \c PccEntry.
 */
typedef struct PitEntry
{
  Packet* npkt; ///< representative Interest packet

  MinTmr timeout; ///< timeout timer
  TscTime expiry; ///< last DN expiration time

  uint8_t nCanBePrefix; ///< how many DNs want CanBePrefix?
  bool mustBeFresh;     ///< entry for MustBeFresh 0 or 1?

  uint8_t lastDnIndex; ///< most recent DN index

  PitDn dns[PIT_ENTRY_MAX_DNS];
  PitUp ups[PIT_ENTRY_MAX_UPS];
} PitEntry;

/** \brief Initialize a PIT entry.
 */
static void
PitEntry_Init(PitEntry* entry, Packet* npkt)
{
  PInterest* interest = Packet_GetInterestHdr(npkt);
  entry->npkt = npkt;
  MinTmr_Init(&entry->timeout);
  entry->expiry = 0;
  entry->nCanBePrefix = interest->canBePrefix;
  entry->mustBeFresh = interest->mustBeFresh;
  entry->dns[0].face = FACEID_INVALID;
  entry->ups[0].face = FACEID_INVALID;
}

/** \brief Finalize a PIT entry.
 */
static void
PitEntry_Finalize(PitEntry* entry)
{
  if (likely(entry->npkt != NULL)) {
    rte_pktmbuf_free(Packet_ToMbuf(entry->npkt));
  }
  MinTmr_Cancel(&entry->timeout);
}

/** \brief Represent PIT entry as a string for debug purpose.
 *  \return A string from thread-local buffer.
 *  \warning Subsequent *ToDebugString calls on the same thread overwrite the buffer.
 */
const char* PitEntry_ToDebugString(PitEntry* entry);

/** \brief Refresh DN record for RX Interest.
 *  \param entry PIT entry, must be initialized.
 *  \param npkt received Interest; will take ownership unless returning -1.
 *  \return index of DN record, or -1 if no slot is available.
 */
int PitEntry_DnRxInterest(Pit* pit, PitEntry* entry, Packet* npkt);

/** \brief Prepare TX Interest to upstream.
 *  \param entry PIT entry, must be initialized.
 *  \param face upstream face.
 *  \param[out] npkt Interest packet, may be NULL if allocation fails.
 *  \return index of UP record, or -1 if no slot is available.
 */
int PitEntry_UpTxInterest(Pit* pit, PitEntry* entry, FaceId face,
                          Packet** npkt);

#endif // NDN_DPDK_CONTAINER_PCCT_PIT_ENTRY_H
