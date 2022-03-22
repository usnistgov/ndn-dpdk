#ifndef NDNDPDK_STRATEGYAPI_PIT_H
#define NDNDPDK_STRATEGYAPI_PIT_H

/** @file */

#include "../pcct/pit-const.h"
#include "common.h"

typedef struct SgPitDn
{
  TscTime expiry;
  char a_[4];
  FaceID face;
} __rte_cache_aligned SgPitDn;

typedef struct SgPitUp
{
  char a_[4];
  FaceID face;
  char b_[1];
  uint8_t nack;

  TscTime lastTx;
  TscDuration suppress;
  uint16_t nTx;
} __rte_cache_aligned SgPitUp;

typedef struct SgPitEntryExt SgPitEntryExt;

typedef struct SgPitEntry
{
  uint8_t a_[48];
  SgPitEntryExt* ext;
  SgPitDn dns[PitMaxDns];
  SgPitUp ups[PitMaxUps];
  uint64_t scratch[PitScratchSize / 8];
} SgPitEntry;

struct SgPitEntryExt
{
  SgPitDn dns[PitMaxExtDns];
  SgPitUp ups[PitMaxExtUps];
  SgPitEntryExt* next;
};

#endif // NDNDPDK_STRATEGYAPI_PIT_H
