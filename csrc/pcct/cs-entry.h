#ifndef NDNDPDK_PCCT_CS_ENTRY_H
#define NDNDPDK_PCCT_CS_ENTRY_H

/** @file */

#include "cs-struct.h"

enum
{
  CsMaxIndirects = 4,
};

typedef struct CsEntry CsEntry;

/**
 * @brief A CS entry.
 *
 * This struct is enclosed in @c PccEntry.
 */
struct CsEntry
{
  CsNode* prev;
  CsNode* next;

  union
  {
    /**
     * @brief The Data packet.
     * @pre Valid if entry is direct.
     */
    Packet* data;

    /**
     * @brief The direct entry.
     * @pre Valid if entry is indirect.
     */
    CsEntry* direct;
  };

  /**
   * @brief When Data becomes non-fresh.
   * @pre Valid if entry is direct.
   */
  TscTime freshUntil;

  /**
   * @brief Count of indirect entries depending on this direct entry,
   *        or -1 to indicate this entry is indirect.
   *
   * A 'direct' CS entry sits in a @c PccEntry of the enclosed Data packet's
   * exact name. When a Data packet is retrieved with an Interest of a prefix
   * name, an additional 'indirect' CS entry is also placed in a @c PccEntry
   * of the prefix name, so that future Interests carrying either the exact
   * name or the same prefix name could find the CS entry.
   */
  int8_t nIndirects;

  CsListID arcList : 8;

  /**
   * @brief Associated indirect entries.
   * @pre Valid if entry is indirect.
   */
  CsEntry* indirect[CsMaxIndirects];
};
static_assert(CsMaxIndirects < INT8_MAX, "");

__attribute__((nonnull)) static __rte_always_inline bool
CsEntry_IsDirect(CsEntry* entry)
{
  return entry->nIndirects >= 0;
}

__attribute__((nonnull)) static __rte_always_inline CsEntry*
CsEntry_GetDirect(CsEntry* entry)
{
  return likely(CsEntry_IsDirect(entry)) ? entry : entry->direct;
}

__attribute__((nonnull)) static __rte_always_inline Packet*
CsEntry_GetData(CsEntry* entry)
{
  return CsEntry_GetDirect(entry)->data;
}

/** @brief Determine if @p entry is fresh. */
__attribute__((nonnull)) static __rte_always_inline bool
CsEntry_IsFresh(CsEntry* entry, TscTime now)
{
  return CsEntry_GetDirect(entry)->freshUntil > now;
}

/** @brief Release enclosed Data packet on a direct entry. */
__attribute__((nonnull)) static inline void
CsEntry_ClearData(CsEntry* entry)
{
  NDNDPDK_ASSERT(CsEntry_IsDirect(entry));
  if (likely(entry->data != NULL)) {
    rte_pktmbuf_free(Packet_ToMbuf(entry->data));
    entry->data = NULL;
  }
}

/** @brief Associate an indirect entry. */
__attribute__((nonnull)) static inline bool
CsEntry_Assoc(CsEntry* indirect, CsEntry* direct)
{
  NDNDPDK_ASSERT(indirect->nIndirects == 0);
  NDNDPDK_ASSERT(CsEntry_IsDirect(direct));

  if (unlikely(direct->nIndirects >= CsMaxIndirects)) {
    return false;
  }

  direct->indirect[direct->nIndirects++] = indirect;
  indirect->direct = direct;
  indirect->nIndirects = -1;
  return true;
}

/** @brief Disassociate an indirect entry. */
__attribute__((nonnull)) static inline void
CsEntry_Disassoc(CsEntry* indirect)
{
  NDNDPDK_ASSERT(!CsEntry_IsDirect(indirect));

  CsEntry* direct = indirect->direct;
  NDNDPDK_ASSERT(direct->nIndirects > 0);
  int8_t i = 0;
  for (; i < direct->nIndirects; ++i) {
    if (direct->indirect[i] == indirect) {
      break;
    }
  }
  NDNDPDK_ASSERT(i < direct->nIndirects);
  direct->indirect[i] = direct->indirect[direct->nIndirects - 1];
  --direct->nIndirects;

  indirect->direct = NULL;
  indirect->nIndirects = 0;
}

/** @brief Clear an entry and prepare it for refresh. */
__attribute__((nonnull)) static inline void
CsEntry_Clear(CsEntry* entry)
{
  if (likely(CsEntry_IsDirect(entry))) {
    CsEntry_ClearData(entry);
  } else {
    CsEntry_Disassoc(entry);
  }
}

/**
 * @brief Finalize an entry.
 * @pre If entry is direct, no indirect entry depends on it.
 */
__attribute__((nonnull)) static inline void
CsEntry_Finalize(CsEntry* entry)
{
  NDNDPDK_ASSERT(entry->nIndirects <= 0);
  CsEntry_Clear(entry);
}

#endif // NDNDPDK_PCCT_CS_ENTRY_H
