#ifndef NDNDPDK_PCCT_PIT_STRUCT_H
#define NDNDPDK_PCCT_PIT_STRUCT_H

/** @file */

#include "common.h"

/**
 * @brief The Pending Interest Table (PIT).
 *
 * Pit* is Pcct*.
 */
typedef struct Pit
{
} Pit;

typedef struct PitEntry PitEntry;

/** @brief Callback to handle strategy timer triggers. */
typedef void (*Pit_SgTimerCb)(Pit* pit, PitEntry* entry, void* arg);

/** @brief PCCT private data for PIT. */
typedef struct PitPriv
{
  uint64_t nEntries; ///< current number of entries

  uint64_t nInsert;   ///< how many inserts created a new PIT entry
  uint64_t nFound;    ///< how many inserts found an existing PIT entry
  uint64_t nCsMatch;  ///< how many inserts matched a CS entry
  uint64_t nAllocErr; ///< how many inserts failed due to allocation error

  uint64_t nDataHit;  ///< how many find-by-Data found PIT entry/entries
  uint64_t nDataMiss; ///< how many find-by-Data did not find PIT entry
  uint64_t nNackHit;  ///< how many find-by-Nack found PIT entry
  uint64_t nNackMiss; ///< how many find-by-Nack did not find PIT entry

  MinSched* timeoutSched;
  Pit_SgTimerCb sgTimerCb;
  void* sgTimerCbArg;
} PitPriv;

#endif // NDNDPDK_PCCT_PIT_STRUCT_H
