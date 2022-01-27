#ifndef NDNDPDK_PCCT_CS_ENTRY_H
#define NDNDPDK_PCCT_CS_ENTRY_H

/** @file */

#include "../ndni/packet.h"
#include "cs-struct.h"

extern const char* CsEntryKindString[];

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
     * @pre kind == CsEntryMemory
     */
    Packet* data;

    /**
     * @brief The Data packet.
     * @pre kind == CsEntryDisk
     */
    uint64_t diskSlot;

    /**
     * @brief The direct entry.
     * @pre kind == CsEntryIndirect
     */
    CsEntry* direct;
  };

  /**
   * @brief When Data becomes non-fresh.
   * @pre kind != CsEntryIndirect
   */
  TscTime freshUntil;

  CsEntryKind kind;

  /**
   * @brief Count of indirect entries depending on this direct entry.
   * @pre kind == CsEntryIndirect
   *
   * A 'direct' CS entry sits in a @c PccEntry of the enclosed Data packet's
   * exact name. When a Data packet is retrieved with an Interest of a prefix
   * name, an additional 'indirect' CS entry is also placed in a @c PccEntry
   * of the prefix name, so that future Interests carrying either the exact
   * name or the same prefix name could find the CS entry.
   */
  uint8_t nIndirects;

  CsListID arcList;

  /**
   * @brief Associated indirect entries.
   * @pre kind != CsEntryIndirect
   */
  CsEntry* indirect[CsMaxIndirects];
};
static_assert(CsMaxIndirects < UINT8_MAX, "");

/**
 * @brief Initialize a CS entry.
 */
__attribute__((nonnull)) static inline void
CsEntry_Init(CsEntry* entry)
{
  *entry = (CsEntry){ 0 };
}

/** @brief Retrieve direct entry. */
__attribute__((nonnull)) static __rte_always_inline CsEntry*
CsEntry_GetDirect(CsEntry* entry)
{
  return unlikely(entry->kind == CsEntryIndirect) ? entry->direct : entry;
}

/**
 * @brief Associate an indirect entry.
 * @pre direct->kind == CsEntryMemory || direct->kind == CsEntryDisk
 * @post indirect->kind == CsEntryIndirect
 */
__attribute__((nonnull)) static inline bool
CsEntry_Assoc(CsEntry* indirect, CsEntry* direct)
{
  NDNDPDK_ASSERT(direct->kind != CsEntryIndirect);
  if (unlikely(direct->nIndirects >= CsMaxIndirects)) {
    return false;
  }

  direct->indirect[direct->nIndirects++] = indirect;
  indirect->kind = CsEntryIndirect;
  indirect->direct = direct;
  return true;
}

/** @brief Disassociate an indirect entry. */
__attribute__((nonnull)) static inline void
CsEntry_Disassoc(CsEntry* indirect)
{
  NDNDPDK_ASSERT(indirect->kind == CsEntryIndirect);
  CsEntry* direct = indirect->direct;

  uint8_t i = 0;
  for (; i < direct->nIndirects; ++i) {
    if (direct->indirect[i] == indirect) {
      break;
    }
  }
  NDNDPDK_ASSERT(i < direct->nIndirects);

  direct->indirect[i] = direct->indirect[--direct->nIndirects];
  indirect->kind = CsEntryNone;
}

/** @brief Clear an entry and prepare it for refresh. */
__attribute__((nonnull)) static inline void
CsEntry_Clear(CsEntry* entry)
{
  switch (entry->kind) {
    case CsEntryNone:
      break;
    case CsEntryMemory:
      rte_pktmbuf_free(Packet_ToMbuf(entry->data));
      entry->kind = CsEntryNone;
      break;
    case CsEntryDisk:
      NDNDPDK_ASSERT(false); // not implemented
      break;
    case CsEntryIndirect:
      CsEntry_Disassoc(entry);
      break;
  }
}

/**
 * @brief Finalize an entry.
 * @pre If entry is direct, no indirect entry depends on it.
 */
__attribute__((nonnull)) static inline void
CsEntry_Finalize(CsEntry* entry)
{
  NDNDPDK_ASSERT(entry->kind == CsEntryIndirect || entry->nIndirects == 0);
  CsEntry_Clear(entry);
}

#endif // NDNDPDK_PCCT_CS_ENTRY_H
