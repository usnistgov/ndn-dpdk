#ifndef NDNDPDK_PCCT_CS_STRUCT_H
#define NDNDPDK_PCCT_CS_STRUCT_H

/** @file */

#include "../core/common.h"
#include "cs-enum.h"

typedef struct CsNode CsNode;

/** @brief The prev-next pointers common in CsEntry and CsList. */
struct CsNode
{
  CsNode* prev;
  CsNode* next;
};

/** @brief A doubly linked list within CS. */
typedef struct CsList
{
  CsNode* prev; // back pointer, self if list is empty
  CsNode* next; // front pointer, self if list is empty
  uint32_t count;
  uint32_t capacity; // unused by CsList
} CsList;

/** @brief Lists for Adaptive Replacement Cache (ARC). */
typedef struct CsArc
{
  double c;   // capacity as float
  double p;   // target size of T1
  CsList T1;  // stored entries that appeared once
  CsList B1;  // tracked entries that appeared once
  CsList T2;  // stored entries that appeared more than once
  CsList B2;  // tracked entries that appeared more than once
  CsList Del; // deleted entries
  // B1.capacity is c, the total capacity
  // B2.capacity is 2c, twice the total capacity
  // T1.capacity is (uint32_t)p
  // T2.capacity is MAX(1, (uint32_t)p)
  // Del.capacity is unused
} CsArc;

typedef struct DiskStore DiskStore;
typedef struct DiskAlloc DiskAlloc;

/**
 * @brief The Content Store (CS).
 *
 * This is embedded in @c Pcct struct.
 */
typedef struct Cs
{
  CsArc direct;    ///< ARC lists of direct entries
  CsList indirect; ///< LRU list of indirect entries

  DiskStore* diskStore;
  DiskAlloc* diskAlloc;
} Cs;

#endif // NDNDPDK_PCCT_CS_STRUCT_H
