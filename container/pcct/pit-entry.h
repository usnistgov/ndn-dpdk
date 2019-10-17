#ifndef NDN_DPDK_CONTAINER_PCCT_PIT_ENTRY_H
#define NDN_DPDK_CONTAINER_PCCT_PIT_ENTRY_H

/// \file

#include "../fib/fib.h"
#include "pit-dn.h"
#include "pit-struct.h"
#include "pit-up.h"

#define PIT_ENTRY_MAX_DNS 6
#define PIT_ENTRY_MAX_UPS 2
#define PIT_ENTRY_EXT_MAX_DNS 16
#define PIT_ENTRY_EXT_MAX_UPS 8
#define PIT_ENTRY_SG_SCRATCH 64

#define PIT_ENTRY_FIBPREFIXL_NBITS_ 9
static_assert((1 << PIT_ENTRY_FIBPREFIXL_NBITS_) > FIB_ENTRY_MAX_NAME_LEN, "");

typedef struct PitEntryExt PitEntryExt;

/** \brief A PIT entry.
 *
 *  This struct is enclosed in \c PccEntry.
 */
struct PitEntry
{
  Packet* npkt;   ///< representative Interest packet
  MinTmr timeout; ///< timeout timer
  TscTime expiry; ///< when all DNs expire

  uint64_t fibPrefixHash; ///< hash value of FIB prefix
  uint32_t fibSeqNum;     ///< FIB entry sequence number
  uint8_t nCanBePrefix;   ///< how many DNs want CanBePrefix?
  uint8_t txHopLimit;     ///< HopLimit for outgoing Interests
  uint16_t fibPrefixL
    : PIT_ENTRY_FIBPREFIXL_NBITS_; ///< TLV-LENGTH of FIB prefix
  bool mustBeFresh : 1;            ///< entry for MustBeFresh 0 or 1?
  bool hasSgTimer : 1; ///< whether timeout is set by strategy or expiry

  PitEntryExt* ext;
  PitDn dns[PIT_ENTRY_MAX_DNS];
  PitUp ups[PIT_ENTRY_MAX_UPS];

  char sgScratch[PIT_ENTRY_SG_SCRATCH];
};
static_assert(offsetof(PitEntry, dns) <= RTE_CACHE_LINE_SIZE, "");

struct PitEntryExt
{
  PitDn dns[PIT_ENTRY_EXT_MAX_DNS];
  PitUp ups[PIT_ENTRY_EXT_MAX_UPS];
  PitEntryExt* next;
};

static inline void
PitEntry_SetFibEntry_(PitEntry* entry,
                      PInterest* interest,
                      const FibEntry* fibEntry)
{
  entry->fibPrefixL = fibEntry->nameL;
  entry->fibSeqNum = fibEntry->seqNum;
  Name* name = &interest->name;
  if (unlikely(interest->activeFh >= 0)) {
    name = &interest->activeFhName;
  }
  entry->fibPrefixHash =
    PName_ComputePrefixHash(&name->p, name->v, fibEntry->nComps);
  memset(entry->sgScratch, 0, PIT_ENTRY_SG_SCRATCH);
}

/** \brief Initialize a PIT entry.
 *  \param npkt the Interest packet.
 */
static inline void
PitEntry_Init(PitEntry* entry, Packet* npkt, const FibEntry* fibEntry)
{
  PInterest* interest = Packet_GetInterestHdr(npkt);
  entry->npkt = npkt;
  MinTmr_Init(&entry->timeout);
  entry->expiry = 0;

  entry->nCanBePrefix = interest->canBePrefix;
  entry->txHopLimit = 0;
  entry->mustBeFresh = interest->mustBeFresh;

  entry->dns[0].face = FACEID_INVALID;
  entry->ups[0].face = FACEID_INVALID;
  entry->ext = NULL;

  PitEntry_SetFibEntry_(entry, interest, fibEntry);
}

/** \brief Finalize a PIT entry.
 */
static inline void
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
const char*
PitEntry_ToDebugString(PitEntry* entry);

/** \brief Reference FIB entry from PIT entry, clear scratch if FIB entry changed.
 *  \param npkt the Interest packet.
 */
static inline void
PitEntry_RefreshFibEntry(PitEntry* entry,
                         Packet* npkt,
                         const FibEntry* fibEntry)
{
  if (likely(entry->fibSeqNum == fibEntry->seqNum)) {
    return;
  }

  PInterest* interest = Packet_GetInterestHdr(npkt);
  PitEntry_SetFibEntry_(entry, interest, fibEntry);
}

/** \brief Retrieve FIB entry via PIT entry's FIB reference.
 *  \pre Calling thread holds rcu_read_lock, which must be retained until it stops
 *       using the returned entry.
 */
FibEntry*
PitEntry_FindFibEntry(PitEntry* entry, Fib* fib);

/** \brief Set timer to erase PIT entry when its last PitDn expires.
 */
void
PitEntry_SetExpiryTimer(PitEntry* entry, Pit* pit);

/** \brief Set timer to invoke strategy after \p after.
 *  \retval Timer set successfully.
 *  \retval Unable to set timer; reverted to expiry timer.
 */
bool
PitEntry_SetSgTimer(PitEntry* entry, Pit* pit, TscDuration after);

void
PitEntry_Timeout_(MinTmr* tmr, void* pit0);

/** \brief Find duplicate nonce among DN records other than \p rxFace.
 *  \return FaceId of PitDn with duplicate nonce, or \c FACEID_INVALID if none.
 */
FaceId
PitEntry_FindDuplicateNonce(PitEntry* entry, uint32_t nonce, FaceId rxFace);

/** \brief Insert new DN record, or update existing DN record.
 *  \param entry PIT entry, must be initialized.
 *  \param npkt received Interest; will take ownership unless returning NULL.
 *  \return DN record, or NULL if no slot is available.
 */
PitDn*
PitEntry_InsertDn(PitEntry* entry, Pit* pit, Packet* npkt);

/** \brief Find existing UP record, or reserve slot for new UP record.
 *  \param entry PIT entry, must be initialized.
 *  \param face upstream face.
 *  \return UP record, or NULL if no slot is available.
 *  \note If returned UP record is unused (no \c PitUp_RecordTx invocation),
 *        it will be overwritten on the next \c PitEntry_ReserveUp invocation.
 */
PitUp*
PitEntry_ReserveUp(PitEntry* entry, Pit* pit, FaceId face);

/** \brief Calculate InterestLifetime for TX Interest.
 *  \return InterestLifetime in millis.
 */
static inline uint32_t
PitEntry_GetTxInterestLifetime(PitEntry* entry, TscTime now)
{
  return TscDuration_ToMillis(entry->expiry - now);
}

/** \brief Calculate HopLimit for TX Interest.
 */
static inline uint8_t
PitEntry_GetTxInterestHopLimit(PitEntry* entry)
{
  return entry->txHopLimit;
}

#endif // NDN_DPDK_CONTAINER_PCCT_PIT_ENTRY_H
