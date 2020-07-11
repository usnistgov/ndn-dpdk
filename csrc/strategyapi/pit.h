#ifndef NDN_DPDK_STRATEGYAPI_PIT_H
#define NDN_DPDK_STRATEGYAPI_PIT_H

/** @file */

#include "common.h"

typedef struct SgPitDn
{
  TscTime expiry;
  char _a[12];
  FaceID face;
} __rte_aligned(32) SgPitDn;

typedef struct SgPitUp
{
  char _a[4];
  FaceID face;
  char _b[1];
  uint8_t nack;

  TscTime lastTx;
  TscDuration suppress;
  uint16_t nTx;
} __rte_aligned(64) SgPitUp;

#define SG_PIT_ENTRY_MAX_DNS 6
#define SG_PIT_ENTRY_MAX_UPS 2
#define SG_PIT_ENTRY_EXT_MAX_DNS 16
#define SG_PIT_ENTRY_EXT_MAX_UPS 8
#define SG_PIT_ENTRY_SCRATCH 64

typedef struct SgPitEntryExt SgPitEntryExt;

typedef struct SgPitEntry
{
  char _a[48];
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

#endif // NDN_DPDK_STRATEGYAPI_PIT_H
