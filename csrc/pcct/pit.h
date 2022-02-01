#ifndef NDNDPDK_PCCT_PIT_H
#define NDNDPDK_PCCT_PIT_H

/** @file */

#include "pcct.h"
#include "pit-result.h"

/** @brief Maximum PIT entry lifetime (millis). */
#define PIT_MAX_LIFETIME 120000

/** @brief Constructor. */
__attribute__((nonnull)) void
Pit_Init(Pit* pit);

/** @brief Trigger expired timers. */
__attribute__((nonnull)) static inline void
Pit_TriggerTimers(Pit* pit)
{
  MinSched_Trigger(pit->timeoutSched);
}

/** @brief Set callback when strategy timer expires. */
__attribute__((nonnull(1))) void
Pit_SetSgTimerCb(Pit* pit, Pit_SgTimerCb cb, void* arg);

/**
 * @brief Insert or find a PIT entry for the given Interest.
 * @param npkt Interest packet.
 *
 * The PIT-CS lookup includes forwarding hint. PInterest's @c activeFh field
 * indicates which fwhint is in use, and setting it to -1 disables fwhint.
 *
 * If there is a CS match, return the CS entry. If there is a PIT match,
 * return the PIT entry. Otherwise, unless the PCCT is full, insert and
 * initialize a PIT entry.
 *
 * When a new PIT entry is inserted, the PIT entry owns @p npkt but does not
 * free it, so the caller may continue using it until @c PitEntry_InsertDn.
 */
__attribute__((nonnull)) PitInsertResult
Pit_Insert(Pit* pit, Packet* npkt, const FibEntry* fibEntry);

/**
 * @brief Erase a PIT entry.
 * @post @p entry is no longer valid.
 */
__attribute__((nonnull)) void
Pit_Erase(Pit* pit, PitEntry* entry);

/** @brief Erase both PIT entries on a PccEntry but retain the PccEntry. */
__attribute__((nonnull)) void
Pit_RawErase01_(Pit* pit, PccEntry* pccEntry);

/**
 * @brief Find PIT entries matching a Data.
 * @param npkt Data packet.
 * @param token PCC token of the packet.
 */
__attribute__((nonnull)) PitFindResult
Pit_FindByData(Pit* pit, Packet* npkt, uint64_t token);

/**
 * @brief Find PIT entry matching a Nack.
 * @param npkt Nack packet.
 * @param token PCC token of the packet.
 */
__attribute__((nonnull)) PitEntry*
Pit_FindByNack(Pit* pit, Packet* npkt, uint64_t token);

__attribute__((nonnull)) static inline uint64_t
PitEntry_GetToken(PitEntry* entry)
{
  // Declaration is in pit-entry.h.
  PccEntry* pccEntry = PccEntry_FromPitEntry(entry);
  NDNDPDK_ASSERT(pccEntry->hasToken);
  return pccEntry->token;
}

#endif // NDNDPDK_PCCT_PIT_H
