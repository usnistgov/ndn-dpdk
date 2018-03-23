#ifndef NDN_DPDK_CONTAINER_PCCT_CS_STRUCT_H
#define NDN_DPDK_CONTAINER_PCCT_CS_STRUCT_H

/// \file

#include "common.h"

typedef struct CsNode CsNode;

/** \brief A node embedded in CsEntry to organize them in a doubly linked list.
 */
struct CsNode
{
  CsNode* prev;
  CsNode* next;
};

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
  uint32_t capacity;
  uint32_t nEntries;

  CsNode head; ///< doubly linked list of entries for cleanup
} CsPriv;

#endif // NDN_DPDK_CONTAINER_PCCT_CS_STRUCT_H
