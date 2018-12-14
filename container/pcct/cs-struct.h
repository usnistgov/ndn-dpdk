#ifndef NDN_DPDK_CONTAINER_PCCT_CS_STRUCT_H
#define NDN_DPDK_CONTAINER_PCCT_CS_STRUCT_H

/// \file

#include "common.h"

typedef struct CsNode CsNode;
typedef struct CsEntry CsEntry;

/** \brief A node embedded in CsEntry to organize them in a doubly linked list.
 */
struct CsNode
{
  CsNode* prev;
  CsNode* next;
};

static CsEntry*
CsNode_AsEntry(CsNode* node)
{
  return (CsEntry*)node;
}

/** \brief A doubly linked list within CS.
 */
typedef struct CsList
{
  CsNode* prev; // back pointer, self if list is empty
  CsNode* next; // front pointer, self if list is empty
  uint32_t count;
  uint32_t capacity; // unused by CsList
} CsList;

void CsList_Init(CsList* csl);

/** \brief Append an entry to back of list.
 */
void CsList_Append(CsList* csl, CsEntry* entry);

/** \brief Remove an entry from the list.
 */
void CsList_Remove(CsList* csl, CsEntry* entry);

/** \brief Move an entry to back of list.
 */
void CsList_MoveToLast(CsList* csl, CsEntry* entry);

typedef void (*CsList_EvictCb)(void* arg, CsEntry* entry);

/** \brief Evict up to \p max entries from front of list.
 *  \param cb callback to erase an entry; the callback must not invoke CsList_Remove.
 */
uint32_t CsList_EvictBulk(CsList* csl, uint32_t max, CsList_EvictCb cb,
                          void* cbarg);

/** \brief Identify a list within CS.
 */
typedef enum CsListId {
  CSL_MD, ///< in-memory direct entries
  CSL_MI, ///< in-memory indirect entries
} CsListId;

const char* CsListId_GetName(CsListId cslId);

/** \brief The Content Store (CS).
 *
 *  Cs* is Pcct*.
 */
typedef struct Cs
{
} Cs;

/** \brief PCCT private data for CS.
 */
typedef struct CsPriv
{
  CsList directFifo;   ///< FIFO list of direct entries
  CsList indirectFifo; ///< FIFO list of indirect entries
} CsPriv;

/** \brief CS size or capacity.
 */
typedef struct CsSize
{
  uint32_t nDirect;   ///< number of direct entries
  uint32_t nIndirect; ///< number of indirect entries
} CsSize;

#endif // NDN_DPDK_CONTAINER_PCCT_CS_STRUCT_H
