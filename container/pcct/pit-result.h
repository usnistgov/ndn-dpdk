#ifndef NDN_DPDK_CONTAINER_PCCT_PIT_RESULT_H
#define NDN_DPDK_CONTAINER_PCCT_PIT_RESULT_H

/// \file

#include "pcc-entry.h"

typedef enum PitResultKind {
  PIT_INSERT_FULL = 0, ///< PIT is full, cannot insert
  PIT_INSERT_PIT0 = 1, ///< created or found PIT entry of MustBeFresh=0
  PIT_INSERT_PIT1 = 2, ///< created or found PIT entry of MustBeFresh=1
  PIT_INSERT_CS = 3,   ///< found existing CS entry that matches the Interest

  PIT_FIND_NONE = 0,  ///< no PIT match
  PIT_FIND_PIT0 = 1,  ///< matched PIT entry of MustBeFresh=0
  PIT_FIND_PIT1 = 2,  ///< matched PIT entry of MustBeFresh=1
  PIT_FIND_PIT01 = 3, ///< matched both PIT entries

  __PIT_RESULT_KIND_MASK = 3,
  __PIT_RESULT_ENTRY_MASK = ~__PIT_RESULT_KIND_MASK,
} PitResultKind;

/** \brief Result of PIT insert/find.
 */
typedef struct PitResult
{
  uintptr_t ptr; ///< PccEntry* | PitResultKind
} PitResult;

static PitResultKind
PitResult_GetKind(PitResult res)
{
  return (PitResultKind)(res.ptr & __PIT_RESULT_KIND_MASK);
}

static PccEntry*
__PitResult_GetPccEntry(PitResult res)
{
  return (PccEntry*)(res.ptr & __PIT_RESULT_ENTRY_MASK);
}

static PitResult
__PitResult_New(PccEntry* entry, PitResultKind kind)
{
  PitResult res = {.ptr = ((uintptr_t)entry | kind) };
  assert(__PitResult_GetPccEntry(res) == entry);
  assert(PitResult_GetKind(res) == kind);
  return res;
}

static PitEntry*
PitInsertResult_GetPitEntry(PitResult res)
{
  PccEntry* entry = __PitResult_GetPccEntry(res);
  switch (PitResult_GetKind(res)) {
    case PIT_INSERT_PIT0:
      return &entry->pitEntry0;
    case PIT_INSERT_PIT1:
      return &entry->pitEntry1;
  }
  assert(false);
}

static CsEntry*
PitInsertResult_GetCsEntry(PitResult res)
{
  assert(PitResult_GetKind(res) == PIT_INSERT_CS);
  PccEntry* entry = __PitResult_GetPccEntry(res);
  return &entry->csEntry;
}

static PitResultKind
__PitFindResult_DetermineKind(PccEntry* entry)
{
  return (entry->hasPitEntry0) | (entry->hasPitEntry1 << 1);
}

static PInterest*
__PitFindResult_GetInterest2(PccEntry* entry, PitResultKind kind)
{
  assert(kind != PIT_FIND_NONE);
  PitEntry* pitEntry =
    (kind & PIT_FIND_PIT0) != 0 ? &entry->pitEntry0 : &entry->pitEntry1;
  return Packet_GetInterestHdr(pitEntry->npkt);
}

/** \brief Get a representative Interest from either PIT entry.
 *  \pre PitResult_GetKind(res) != PIT_FIND_NONE
 */
static PInterest*
__PitFindResult_GetInterest(PitResult res)
{
  PccEntry* entry = __PitResult_GetPccEntry(res);
  return __PitFindResult_GetInterest2(entry, PitResult_GetKind(res));
}

static PitEntry*
PitFindResult_GetPitEntry0(PitResult res)
{
  if ((PitResult_GetKind(res) & PIT_FIND_PIT0) == 0) {
    return NULL;
  }
  PccEntry* entry = __PitResult_GetPccEntry(res);
  return &entry->pitEntry0;
}

static PitEntry*
PitFindResult_GetPitEntry1(PitResult res)
{
  if ((PitResult_GetKind(res) & PIT_FIND_PIT1) == 0) {
    return NULL;
  }
  PccEntry* entry = __PitResult_GetPccEntry(res);
  return &entry->pitEntry1;
}

#endif // NDN_DPDK_CONTAINER_PCCT_PIT_RESULT_H
