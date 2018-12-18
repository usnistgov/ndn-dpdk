#ifndef NDN_DPDK_CONTAINER_PCCT_CS_STRUCT_H
#define NDN_DPDK_CONTAINER_PCCT_CS_STRUCT_H

/// \file

#include "common.h"

/** \brief prev-next pointers common in CsEntry and CsList.
 */
typedef struct CsNode CsNode;

/** \brief A doubly linked list within CS.
 */
typedef struct CsList
{
  CsNode* prev; // back pointer, self if list is empty
  CsNode* next; // front pointer, self if list is empty
  uint32_t count;
  uint32_t capacity; // unused by CsList
} CsList;

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
  CsList directFifo;  ///< FIFO list of direct entries
  CsList indirectLru; ///< LRU list of indirect entries
} CsPriv;

#endif // NDN_DPDK_CONTAINER_PCCT_CS_STRUCT_H
