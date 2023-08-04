#ifndef NDNDPDK_PCCT_PIT_DN_UP_IT_H
#define NDNDPDK_PCCT_PIT_DN_UP_IT_H

/** @file */

#include "pit-entry.h"

typedef struct PitDnUpIt_ {
  union {
    void* current;
    PitDn* dn; ///< current PitDn
    PitUp* up; ///< current PitUp
  };
  int index; ///< index of PitDn/PitUp

  int i;   ///< (pvt) index within this array
  int max; ///< (pvt) upper bound of this array
  union {
    void* array; // (pvt) start of array
    PitDn* dns;
    PitUp* ups;
  };

  PitEntryExt** nextPtr;     ///< (pvt) next extension
  struct PccEntry* pccEntry; ///< (pvt) PCC entry for obtaining mempool
} PitDnUpIt_;

__attribute__((nonnull)) static inline PitDnUpIt_
PitDnUpIt_New_(PitEntry* entry, int maxInEntry, void* array) {
  return (PitDnUpIt_){
    current: array,
    max: maxInEntry,
    array: array,
    nextPtr: &entry->ext,
    pccEntry: entry->pccEntry,
  };
}

__attribute__((nonnull)) bool
PitDnUpIt_Extend_(PitDnUpIt_* it, int maxInExt, size_t offsetInExt);

__attribute__((nonnull)) static inline bool
PitDnUpIt_Valid_(PitDnUpIt_* it, bool canExtend, int maxInExt, size_t offsetInExt) {
  if (likely(it->i < it->max)) {
    return true;
  }
  return canExtend && PitDnUpIt_Extend_(it, maxInExt, offsetInExt);
}

__attribute__((nonnull)) static inline void
PitDnUpIt_EnterExt_(PitDnUpIt_* it, PitEntryExt* ext, int maxInExt, size_t offsetInExt) {
  it->i = 0;
  it->max = maxInExt;
  it->array = RTE_PTR_ADD(ext, offsetInExt);
  it->current = it->array;
  it->nextPtr = &ext->next;
}

__attribute__((nonnull)) static inline void
PitDnUpIt_Next_(PitDnUpIt_* it, size_t sizeofRecord, int maxInExt, size_t offsetInExt) {
  NDNDPDK_ASSERT(it->i < it->max);
  ++it->index;
  ++it->i;
  it->current = RTE_PTR_ADD(it->current, sizeofRecord);
  if (likely(it->i < it->max)) {
    return;
  }

  PitEntryExt* ext = *it->nextPtr;
  if (ext == NULL) {
    return;
  }
  PitDnUpIt_EnterExt_(it, ext, maxInExt, offsetInExt);
}

/** @brief Iterator of DN slots in PIT entry. */
typedef PitDnUpIt_ PitDnIt;

/**
 * @brief Mark current DN slot is used.
 * @pre This and subsequent DN slots are unused.
 * @post This DN slot is used, subsequent DN slots are unused.
 */
__attribute__((nonnull)) static inline void
PitDn_UseSlot(PitDnIt* it) {
  int next = it->i + 1;
  if (likely(next < it->max)) {
    it->dns[next].face = 0;
  }
}

/**
 * @brief Iterate over DN slots in PIT entry.
 *
 * @code
 * PitDn_Each(it, pitEntry, false) {
 *   int index = it.index;
 *   PitDn* dn = it.dn;
 *   if (dn->face == 0) { // reaching the end of used slots
 *     break;
 *   }
 *   // access the DN record
 * }
 * @endcode
 *
 * @code
 * PitDn_Each(it, pitEntry, true) {
 *   int index = it.index;
 *   PitDn* dn = it.dn;
 *   if (dn->face == faceID) { // found existing slot with wanted face
 *   }
 *   if (dn->face == 0) { // reaching the end of used slots
 *     PitDn_UseSlot(&it); // claim this slot as used
 *     dn->face = faceID;
 *     // initialized the slot
 *   }
 * }
 * @endcode
 */
#define PitDn_Each(var, entry, canExtend)                                                          \
  for (PitDnIt var = PitDnUpIt_New_((entry), PitMaxDns, (entry)->dns);                             \
       PitDnUpIt_Valid_(&var, (canExtend), PitMaxExtDns, offsetof(PitEntryExt, dns));              \
       PitDnUpIt_Next_(&var, sizeof(PitDn), PitMaxExtDns, offsetof(PitEntryExt, dns)))

/** @brief Iterator of UP slots in PIT entry. */
typedef PitDnUpIt_ PitUpIt;

/**
 * @brief Mark current UP slot is used.
 * @pre This and subsequent UP slots are unused.
 * @post This UP slot is used, subsequent UP slots are unused.
 */
__attribute__((nonnull)) static inline void
PitUp_UseSlot(PitUpIt* it) {
  int next = it->i + 1;
  if (likely(next < it->max)) {
    it->ups[next].face = 0;
  }
}

/**
 * @brief Iterate over DN slots in PIT entry.
 * @sa PitDn_Each
 *
 * @code
 * PitUp_Each(it, pitEntry, false) {
 *   int index = it.index;
 *   PitUp* up = it.up;
 * }
 * @endcode
 */
#define PitUp_Each(var, entry, canExtend)                                                          \
  for (PitUpIt var = PitDnUpIt_New_((entry), PitMaxUps, (entry)->ups);                             \
       PitDnUpIt_Valid_(&var, (canExtend), PitMaxExtUps, offsetof(PitEntryExt, ups));              \
       PitDnUpIt_Next_(&var, sizeof(PitUp), PitMaxExtUps, offsetof(PitEntryExt, ups)))

#endif // NDNDPDK_PCCT_PIT_DN_UP_IT_H
