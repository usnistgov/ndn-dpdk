#ifndef NDN_DPDK_PCCT_PIT_RESULT_H
#define NDN_DPDK_PCCT_PIT_RESULT_H

/// \file

#include "pcc-entry.h"

/** \brief Private base of PitInsertResult and PitFindResult.
 */
typedef struct PitResult
{
  PccEntry* entry;
  int kind;
} PitResult;

static inline PitResult
PitResult_New_(PccEntry* entry, int kind)
{
  PitResult pr = { .entry = entry, .kind = kind };
  return pr;
}

/** \brief Result of PIT insert.
 */
typedef PitResult PitInsertResult;

/** \brief Result kind of PIT insert.
 */
typedef enum PitInsertResultKind
{
  PIT_INSERT_FULL = 0, ///< PIT is full, cannot insert
  PIT_INSERT_PIT0 = 1, ///< created or found PIT entry of MustBeFresh=0
  PIT_INSERT_PIT1 = 2, ///< created or found PIT entry of MustBeFresh=1
  PIT_INSERT_CS = 3,   ///< found existing CS entry that matches the Interest
} PitInsertResultKind;

static inline PitInsertResultKind
PitInsertResult_GetKind(PitInsertResult res)
{
  return res.kind;
}

static inline PitEntry*
PitInsertResult_GetPitEntry(PitInsertResult res)
{
  switch (res.kind) {
    case PIT_INSERT_PIT0:
      return PccEntry_GetPitEntry0(res.entry);
    case PIT_INSERT_PIT1:
      return PccEntry_GetPitEntry1(res.entry);
    default:
      assert(false);
      return NULL;
  }
}

static inline CsEntry*
PitInsertResult_GetCsEntry(PitInsertResult res)
{
  assert(res.kind == PIT_INSERT_CS);
  return PccEntry_GetCsEntry(res.entry);
}

/** \brief Result of PIT find.
 */
typedef PitResult PitFindResult;

/** \brief Result flag of PIT find, bitwise OR.
 */
typedef enum PitFindResultFlag
{
  PIT_FIND_NONE = 0, ///< no PIT match

  PIT_FIND_PIT0 = (1 << 0), ///< matched PIT entry of MustBeFresh=0
  PIT_FIND_PIT1 = (1 << 1), ///< matched PIT entry of MustBeFresh=1

  /// need Data digest to determine match, PccEntry is set on PitInsertResult,
  /// PIT_FIND_PIT0 and PIT_FIND_PIT1 indicate existence of PIT entries.
  PIT_FIND_NEED_DIGEST = (1 << 2),
} PitFindResultFlag;

static inline bool
PitFindResult_Is(PitFindResult res, PitFindResultFlag flag)
{
  if (flag == PIT_FIND_NONE) {
    return res.kind == PIT_FIND_NONE;
  }
  return (res.kind & flag) != 0;
}

static inline PitEntry*
PitFindResult_GetPitEntry0(PitFindResult res)
{
  if (!PitFindResult_Is(res, PIT_FIND_PIT0)) {
    return NULL;
  }
  return PccEntry_GetPitEntry0(res.entry);
}

static inline PitEntry*
PitFindResult_GetPitEntry1(PitFindResult res)
{
  if (!PitFindResult_Is(res, PIT_FIND_PIT1)) {
    return NULL;
  }
  return PccEntry_GetPitEntry1(res.entry);
}

/** \brief Get a representative Interest from either PIT entry.
 */
static inline PInterest*
PitFindResult_GetInterest_(PitFindResult res)
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

#endif // NDN_DPDK_PCCT_PIT_RESULT_H
