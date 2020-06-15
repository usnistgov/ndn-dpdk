#ifndef NDN_DPDK_PCCT_CS_STRUCT_H
#define NDN_DPDK_PCCT_CS_STRUCT_H

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

/** \brief Lists for Adaptive Replacement Cache (ARC).
 */
typedef struct CsArc
{
  double c;   // capacity as float
  double p;   // target size of T1
  CsList T1;  // stored entries that appeared once
  CsList B1;  // tracked entries that appeared once
  CsList T2;  // stored entries that appeared more than once
  CsList B2;  // tracked entries that appeared more than once
  CsList DEL; // deleted entries
  // B1.capacity is c, the total capacity
  // B2.capacity is 2c, twice the total capacity
  // T1.capacity is (uint32_t)p
  // T2.capacity is MAX(1, (uint32_t)p)
  // DEL.capacity is unused
} CsArc;

typedef enum CsArcListId
{
  CSL_ARC_NONE,
  CSL_ARC_T1,
  CSL_ARC_B1,
  CSL_ARC_T2,
  CSL_ARC_B2,
  CSL_ARC_DEL,
} CsArcListId;

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
  CsArc directArc;    ///< ARC lists of direct entries
  CsList indirectLru; ///< LRU list of indirect entries
} CsPriv;

#endif // NDN_DPDK_PCCT_CS_STRUCT_H
