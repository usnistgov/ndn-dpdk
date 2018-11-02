#ifndef NDN_DPDK_CONTAINER_PCCT_PIT_RESULT_H
#define NDN_DPDK_CONTAINER_PCCT_PIT_RESULT_H

/// \file

#include "pcc-entry.h"

/** \brief Private base of PitInsertResult and PitFindResult.
 */
typedef struct PitResult
{
  PccEntry* entry;
  int kind;
} PitResult;

static PitResult
__PitResult_New(PccEntry* entry, int kind)
{
  PitResult pr = {.entry = entry, .kind = kind };
  return pr;
}

/** \brief Result of PIT insert.
 */
typedef PitResult PitInsertResult;

typedef enum PitInsertResultKind {
  PIT_INSERT_FULL = 0, ///< PIT is full, cannot insert
  PIT_INSERT_PIT0 = 1, ///< created or found PIT entry of MustBeFresh=0
  PIT_INSERT_PIT1 = 2, ///< created or found PIT entry of MustBeFresh=1
  PIT_INSERT_CS = 3,   ///< found existing CS entry that matches the Interest
} PitInsertResultKind;

static PitInsertResultKind
PitInsertResult_GetKind(PitInsertResult res)
{
  return res.kind;
}

static PitEntry*
PitInsertResult_GetPitEntry(PitInsertResult res)
{
  switch (res.kind) {
    case PIT_INSERT_PIT0:
      return &res.entry->pitEntry0;
    case PIT_INSERT_PIT1:
      return &res.entry->pitEntry1;
    default:
      assert(false);
      return NULL;
  }
}

static CsEntry*
PitInsertResult_GetCsEntry(PitInsertResult res)
{
  assert(res.kind == PIT_INSERT_CS);
  return &res.entry->csEntry;
}

/** \brief Result of PIT find.
 */
typedef PitResult PitFindResult;

typedef enum PitFindResultKind {
  PIT_FIND_NONE = 0,  ///< no PIT match
  PIT_FIND_PIT0 = 1,  ///< matched PIT entry of MustBeFresh=0
  PIT_FIND_PIT1 = 2,  ///< matched PIT entry of MustBeFresh=1
  PIT_FIND_PIT01 = 3, ///< matched both PIT entries
} PitFindResultKind;

static PitFindResultKind
PitFindResult_GetKind(PitFindResult res)
{
  return res.kind;
}

static PitEntry*
PitFindResult_GetPitEntry0(PitFindResult res)
{
  if ((res.kind & PIT_FIND_PIT0) == 0) {
    return NULL;
  }
  return &res.entry->pitEntry0;
}

static PitEntry*
PitFindResult_GetPitEntry1(PitFindResult res)
{
  if ((res.kind & PIT_FIND_PIT1) == 0) {
    return NULL;
  }
  return &res.entry->pitEntry1;
}

/** \brief Get a representative Interest from either PIT entry.
 */
static PInterest*
__PitFindResult_GetInterest(PitFindResult res)
{
  PitEntry* pitEntry = NULL;
  switch (res.kind) {
    case PIT_FIND_PIT0:
    case PIT_FIND_PIT01:
      pitEntry = &res.entry->pitEntry0;
      break;
    case PIT_FIND_PIT1:
      pitEntry = &res.entry->pitEntry1;
      break;
    default:
      return NULL;
  }
  return Packet_GetInterestHdr(pitEntry->npkt);
}

#endif // NDN_DPDK_CONTAINER_PCCT_PIT_RESULT_H
