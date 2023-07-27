#ifndef NDNDPDK_PCCT_PCC_ENTRY_H
#define NDNDPDK_PCCT_PCC_ENTRY_H

/** @file */

#include "../vendor/uthash-handle.h"
#include "cs-entry.h"
#include "pcc-key.h"
#include "pit-entry.h"

enum {
  PccTokenSize = 6,
  PccTokenBits = PccTokenSize * CHAR_BIT,
  PccTokenMask = ((uint64_t)1 << PccTokenBits) - 1,
};

typedef struct PccEntry PccEntry;

/**
 * @brief PCC entry slot.
 *
 * Each slot has room for either a PIT entry or a CS entry.
 */
typedef union PccSlot {
  struct {
    uint64_t a_[2];
    PccEntry* pccEntry; ///< NULL indicates unoccupied slot
  };
  PitEntry pitEntry;
  CsEntry csEntry;
} PccSlot;

/** @brief Identify a PCC entry slot. */
typedef enum PccSlotIndex {
  PCC_SLOT_NONE = 0,
  PCC_SLOT1 = 1,
  PCC_SLOT2 = 2,
  PCC_SLOT3 = 3,
} PccSlotIndex;

/**
 * @brief PCC entry extension.
 *
 * It adds slot2 and slot3 to a PCC entry.
 */
typedef struct PccEntryExt {
  PccSlot slot2;
  PccSlot slot3;
} PccEntryExt;

/**
 * @brief PIT-CS composite entry.
 *
 * It contains PCC entry index and slot1.
 */
struct PccEntry {
  PccKey key;
  UT_hash_handle hh;

  union {
    struct {
      bool hasToken : 1;
      int : 1;
      PccSlotIndex pitEntry0Slot : 2;
      PccSlotIndex pitEntry1Slot : 2;
      PccSlotIndex csEntrySlot : 2;
      int : 8;
      uint64_t token : PccTokenBits;
    } __rte_packed;
    struct {
      int : 2;
      const int hasPitEntry0 : 2;
      const int hasPitEntry1 : 2;
      const int hasCsEntry : 2;
      uint64_t : 56;
    } __rte_packed;
    struct {
      int : 2;
      const int hasPitEntries : 4;
      uint64_t : 58;
    } __rte_packed;
    struct {
      int : 2;
      const int hasEntries : 6;
      uint64_t : 56;
    } __rte_packed;
    uint64_t tokenQword;
  };
  RTE_MARKER tokenAfter_;

  PccSlot slot1;
  PccEntryExt* ext;
};
static_assert(offsetof(PccEntry, tokenQword) + sizeof(uint64_t) == offsetof(PccEntry, tokenAfter_),
              "");

__attribute__((nonnull)) static __rte_always_inline PccSlot*
PccEntry_GetSlot_(PccEntry* entry, PccSlotIndex slot) {
  switch (slot) {
    case PCC_SLOT1:
      return &entry->slot1;
    case PCC_SLOT2:
      NDNDPDK_ASSERT(entry->ext != NULL);
      return &entry->ext->slot2;
    case PCC_SLOT3:
      NDNDPDK_ASSERT(entry->ext != NULL);
      return &entry->ext->slot3;
    default:
      NDNDPDK_ASSERT(false);
      return NULL;
  }
}

__attribute__((nonnull)) PccSlotIndex
PccEntry_AllocateSlot_(PccEntry* entry, PccSlot** slot);

__attribute__((nonnull)) void
PccEntry_ClearSlot_(PccEntry* entry, PccSlotIndex slot);

/**
 * @brief Get PIT entry of MustBeFresh=0 from @p entry .
 * @pre @c entry->hasPitEntry0
 */
__attribute__((nonnull, returns_nonnull)) static inline PitEntry*
PccEntry_GetPitEntry0(PccEntry* entry) {
  return &PccEntry_GetSlot_(entry, entry->pitEntry0Slot)->pitEntry;
}

/**
 * @brief Add PIT entry of MustBeFresh=0 to @p entry .
 * @return existing or new entry.
 * @retval NULL allocation failure.
 */
__attribute__((nonnull)) static inline PitEntry*
PccEntry_AddPitEntry0(PccEntry* entry) {
  if (entry->hasPitEntry0) {
    return PccEntry_GetPitEntry0(entry);
  }
  PccSlot* slot = NULL;
  entry->pitEntry0Slot = PccEntry_AllocateSlot_(entry, &slot);
  if (unlikely(slot == NULL)) {
    return NULL;
  }
  return &slot->pitEntry;
}

/** @brief Remove PIT entry of MustBeFresh=0 from @p entry . */
__attribute__((nonnull)) static inline void
PccEntry_RemovePitEntry0(PccEntry* entry) {
  PccEntry_ClearSlot_(entry, entry->pitEntry0Slot);
  entry->pitEntry0Slot = PCC_SLOT_NONE;
}

/**
 * @brief Get PIT entry of MustBeFresh=1 from @p entry .
 * @pre @c entry->hasPitEntry1
 */
__attribute__((nonnull, returns_nonnull)) static inline PitEntry*
PccEntry_GetPitEntry1(PccEntry* entry) {
  return &PccEntry_GetSlot_(entry, entry->pitEntry1Slot)->pitEntry;
}

/**
 * @brief Add PIT entry of MustBeFresh=1 to @p entry .
 * @return existing or new entry.
 * @retval NULL allocation failure.
 */
__attribute__((nonnull)) static inline PitEntry*
PccEntry_AddPitEntry1(PccEntry* entry) {
  if (entry->hasPitEntry1) {
    return PccEntry_GetPitEntry1(entry);
  }
  PccSlot* slot = NULL;
  entry->pitEntry1Slot = PccEntry_AllocateSlot_(entry, &slot);
  if (unlikely(slot == NULL)) {
    return NULL;
  }
  return &slot->pitEntry;
}

/** @brief Remove PIT entry of MustBeFresh=1 from @p entry . */
__attribute__((nonnull)) static inline void
PccEntry_RemovePitEntry1(PccEntry* entry) {
  PccEntry_ClearSlot_(entry, entry->pitEntry1Slot);
  entry->pitEntry1Slot = PCC_SLOT_NONE;
}

/**
 * @brief Get CS entry from @p entry .
 * @pre @c entry->hasCsEntry
 */
__attribute__((nonnull, returns_nonnull)) static inline CsEntry*
PccEntry_GetCsEntry(PccEntry* entry) {
  return &PccEntry_GetSlot_(entry, entry->csEntrySlot)->csEntry;
}

/**
 * @brief Add CS entry to @p entry .
 * @return existing or new entry.
 * @retval NULL allocation failure.
 */
__attribute__((nonnull)) static inline CsEntry*
PccEntry_AddCsEntry(PccEntry* entry) {
  if (entry->hasCsEntry) {
    return PccEntry_GetCsEntry(entry);
  }
  PccSlot* slot = NULL;
  entry->csEntrySlot = PccEntry_AllocateSlot_(entry, &slot);
  if (unlikely(slot == NULL)) {
    return NULL;
  }
  return &slot->csEntry;
}

/** @brief Remove CS entry from @p entry . */
__attribute__((nonnull)) static inline void
PccEntry_RemoveCsEntry(PccEntry* entry) {
  PccEntry_ClearSlot_(entry, entry->csEntrySlot);
  entry->csEntrySlot = PCC_SLOT_NONE;
}

#endif // NDNDPDK_PCCT_PCC_ENTRY_H
