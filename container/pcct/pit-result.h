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

/** \brief Result kind of PIT insert.
 */
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

/** \brief Result flag of PIT find, bitwise OR.
 */
typedef enum PitFindResultFlag {
  PIT_FIND_NONE = 0, ///< no PIT match

  PIT_FIND_PIT0 = (1 << 0), ///< matched PIT entry of MustBeFresh=0
  PIT_FIND_PIT1 = (1 << 1), ///< matched PIT entry of MustBeFresh=1

  /// need Data digest to determine match, PccEntry is set on PitInsertResult,
  /// PIT_FIND_PIT0 and PIT_FIND_PIT1 indicate existence of PIT entries.
  PIT_FIND_NEED_DIGEST = (1 << 2),
} PitFindResultFlag;

static bool
PitFindResult_Is(PitFindResult res, PitFindResultFlag flag)
{
  if (flag == PIT_FIND_NONE) {
    return res.kind == PIT_FIND_NONE;
  }
  return (res.kind & flag) != 0;
}

static PitEntry*
PitFindResult_GetPitEntry0(PitFindResult res)
{
  if (!PitFindResult_Is(res, PIT_FIND_PIT0)) {
    return NULL;
  }
  return &res.entry->pitEntry0;
}

static PitEntry*
PitFindResult_GetPitEntry1(PitFindResult res)
{
  if (!PitFindResult_Is(res, PIT_FIND_PIT1)) {
    return NULL;
  }
  return &res.entry->pitEntry1;
}

/** \brief Get a representative Interest from either PIT entry.
 */
static PInterest*
__PitFindResult_GetInterest(PitFindResult res)
{
  PitEntry* pitEntry = PitFindResult_GetPitEntry0(res);
  if (pitEntry == NULL) {
    pitEntry = PitFindResult_GetPitEntry1(res);
  }
  if (pitEntry == NULL) {
    return NULL;
  }
  return Packet_GetInterestHdr(pitEntry->npkt);
}

#endif // NDN_DPDK_CONTAINER_PCCT_PIT_RESULT_H
