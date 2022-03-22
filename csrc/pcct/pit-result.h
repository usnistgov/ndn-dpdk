#ifndef NDNDPDK_PCCT_PIT_RESULT_H
#define NDNDPDK_PCCT_PIT_RESULT_H

/** @file */

#include "pcc-entry.h"

/** @brief Result kind of PIT insert. */
typedef enum PitInsertResultKind
{
  PIT_INSERT_FULL = 0, ///< PIT is full, cannot insert
  PIT_INSERT_PIT = 1,  ///< created or found PIT entry
  PIT_INSERT_CS = 2,   ///< found existing CS entry that matches the Interest
} PitInsertResultKind;

/** @brief Result of PIT insert. */
typedef struct PitInsertResult
{
  PitInsertResultKind kind;
  union
  {
    PitEntry* pitEntry; ///< PIT entry, valid if kind==PIT_INSERT_PIT
    CsEntry* csEntry;   ///< direct CS entry, valid if kind==PIT_INSERT_CS
  };
} PitInsertResult;

/** @brief Result flag of PIT find, bitwise OR. */
typedef enum PitFindResultFlag
{
  PIT_FIND_NONE = 0, ///< no PIT match

  PIT_FIND_PIT0 = RTE_BIT32(0), ///< matched PIT entry of MustBeFresh=0
  PIT_FIND_PIT1 = RTE_BIT32(1), ///< matched PIT entry of MustBeFresh=1

  /// need Data digest to determine match, PccEntry is set on PitInsertResult,
  /// PIT_FIND_PIT0 and PIT_FIND_PIT1 indicate existence of PIT entries.
  PIT_FIND_NEED_DIGEST = RTE_BIT32(2),
} PitFindResultFlag;

/** @brief Result of PIT find. */
typedef struct PitFindResult
{
  PccEntry* entry;
  uint8_t kind;
} PitFindResult;

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

/** @brief Get a representative Interest from either PIT entry. */
static inline PInterest*
PitFindResult_GetInterest(PitFindResult res)
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

#endif // NDNDPDK_PCCT_PIT_RESULT_H
