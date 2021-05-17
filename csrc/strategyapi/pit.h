#ifndef NDNDPDK_STRATEGYAPI_PIT_H
#define NDNDPDK_STRATEGYAPI_PIT_H

/** @file */

#include "common.h"

typedef struct SgPitDn
{
  TscTime expiry;
  char a_[4];
  FaceID face;
} __rte_aligned(64) SgPitDn;

typedef struct SgPitUp
{
  char a_[4];
  FaceID face;
  char b_[1];
  uint8_t nack;

  TscTime lastTx;
  TscDuration suppress;
  uint16_t nTx;
} __rte_aligned(64) SgPitUp;

#define SG_PIT_ENTRY_MAX_DNS 6
#define SG_PIT_ENTRY_MAX_UPS 2
#define SG_PIT_ENTRY_EXT_MAX_DNS 6
#define SG_PIT_ENTRY_EXT_MAX_UPS 4
#define SG_PIT_ENTRY_SCRATCH 64

typedef struct SgPitEntryExt SgPitEntryExt;

typedef struct SgPitEntry
{
  char a_[48];
  SgPitEntryExt* ext;
  SgPitDn dns[SG_PIT_ENTRY_MAX_DNS];
  SgPitUp ups[SG_PIT_ENTRY_MAX_UPS];
  uint64_t scratch[SG_PIT_ENTRY_SCRATCH / 8];
} SgPitEntry;

struct SgPitEntryExt
{
  SgPitDn dns[SG_PIT_ENTRY_EXT_MAX_DNS];
  SgPitUp ups[SG_PIT_ENTRY_EXT_MAX_UPS];
  SgPitEntryExt* next;
};

#endif // NDNDPDK_STRATEGYAPI_PIT_H
