#ifndef NDN_DPDK_CONTAINER_PCCT_PIT_ENTRY_H
#define NDN_DPDK_CONTAINER_PCCT_PIT_ENTRY_H

/// \file

#include "pit-dn.h"
#include "pit-struct.h"
#include "pit-up.h"

#define PIT_ENTRY_MAX_DNS 6
#define PIT_ENTRY_MAX_UPS 2
#define PIT_ENTRY_EXT_MAX_DNS 72
#define PIT_ENTRY_EXT_MAX_UPS 36

typedef struct PitEntryExt PitEntryExt;

/** \brief A PIT entry.
 *
 *  This struct is enclosed in \c PccEntry.
 */
typedef struct PitEntry
{
  Packet* npkt; ///< representative Interest packet

  MinTmr timeout; ///< timeout timer

  TscTime expiry; ///< when all DNs expire

  bool mustBeFresh;     ///< entry for MustBeFresh 0 or 1?
  uint8_t nCanBePrefix; ///< how many DNs want CanBePrefix?
  uint8_t txHopLimit;   ///< HopLimit for outgoing Interests

  PitEntryExt* ext;
  PitDn dns[PIT_ENTRY_MAX_DNS];
  PitUp ups[PIT_ENTRY_MAX_UPS];
} PitEntry;
static_assert(offsetof(PitEntry, dns) <= RTE_CACHE_LINE_SIZE, "");

struct PitEntryExt
{
  PitDn dns[PIT_ENTRY_EXT_MAX_DNS];
  PitUp ups[PIT_ENTRY_EXT_MAX_UPS];
  PitEntryExt* next;
};

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
  entry->ext = NULL;
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
  for (PitEntryExt* ext = entry->ext; unlikely(ext != NULL);) {
    PitEntryExt* next = ext->next;
    rte_mempool_put(rte_mempool_from_obj(ext), ext);
    ext = next;
  }
}

/** \brief Represent PIT entry as a string for debug purpose.
 *  \return A string from thread-local buffer.
 *  \warning Subsequent *ToDebugString calls on the same thread overwrite the buffer.
 */
const char* PitEntry_ToDebugString(PitEntry* entry);

/** \brief Find duplicate nonce among DN records other than \p rxFace.
 *  \return FaceId of PitDn with duplicate nonce, or \c FACEID_INVALID if none.
 */
FaceId PitEntry_FindDuplicateNonce(PitEntry* entry, uint32_t nonce,
                                   FaceId rxFace);

/** \brief Insert new DN record, or update existing DN record.
 *  \param entry PIT entry, must be initialized.
 *  \param npkt received Interest; will take ownership unless returning NULL.
 *  \return DN record, or NULL if no slot is available.
 */
PitDn* PitEntry_InsertDn(PitEntry* entry, Pit* pit, Packet* npkt);

/** \brief Find existing UP record, or reserve slot for new UP record.
 *  \param entry PIT entry, must be initialized.
 *  \param face upstream face.
 *  \return UP record, or NULL if no slot is available.
 *  \note If returned UP record is unused (no \c PitUp_RecordTx invocation),
 *        it will be overwritten on the next \c PitEntry_ReserveUp invocation.
 */
PitUp* PitEntry_ReserveUp(PitEntry* entry, Pit* pit, FaceId face);

/** \brief Calculate InterestLifetime for TX Interest.
 *  \return InterestLifetime in millis.
 */
static uint32_t
PitEntry_GetTxInterestLifetime(PitEntry* entry, TscTime now)
{
  return TscDuration_ToMillis(entry->expiry - now);
}

/** \brief Calculate HopLimit for TX Interest.
 */
static uint8_t
PitEntry_GetTxInterestHopLimit(PitEntry* entry)
{
  return entry->txHopLimit;
}

#endif // NDN_DPDK_CONTAINER_PCCT_PIT_ENTRY_H
