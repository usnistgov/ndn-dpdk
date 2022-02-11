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
  CsNode* prev;      ///< back pointer, self if list is empty
  CsNode* next;      ///< front pointer, self if list is empty
  uint32_t count;    ///< number of entries
  uint32_t capacity; ///< unused by CsList
} CsList;

typedef struct CsEntry CsEntry;

typedef void (*CsArc_MoveCb)(CsEntry* entry, CsListID src, CsListID dst, uintptr_t ctx);

/** @brief Lists for Adaptive Replacement Cache (ARC). */
typedef struct CsArc
{
  double c;   ///< capacity @c c as float
  double p;   ///< target size of T1
  CsList T1;  ///< stored entries that appeared once
  CsList B1;  ///< tracked entries that appeared once
  CsList T2;  ///< stored entries that appeared more than once
  CsList B2;  ///< tracked entries that appeared more than once
  CsList Del; ///< deleted entries

  CsArc_MoveCb moveCb; ///< handler function when entry is moved between lists
  uintptr_t moveCtx;   ///< context argument to @c moveCb
} CsArc;

/** @brief Access @c c as uint32. */
#define CsArc_c(arc) ((arc)->B1.capacity)

/** @brief Access @c 2c as uint32. */
#define CsArc_2c(arc) ((arc)->Del.capacity)

/** @brief Access @c p as uint32. */
#define CsArc_p(arc) ((arc)->T1.capacity)

/** @brief Access @c MAX(p,1) as uint32. */
#define CsArc_p1(arc) ((arc)->T2.capacity)

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

  uint64_t nHitMemory;
  uint64_t nHitDisk;
  uint64_t nHitIndirect;
  uint64_t nDiskInsert;
  uint64_t nDiskDelete;
  uint64_t nDiskFull;
} Cs;

#endif // NDNDPDK_PCCT_CS_STRUCT_H
