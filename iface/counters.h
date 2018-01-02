#ifndef NDN_DPDK_IFACE_COUNTERS_H
#define NDN_DPDK_IFACE_COUNTERS_H

#include "common.h"

typedef struct RxL2Counters
{
  uint64_t nFrames;
  uint64_t nOctets;

  uint64_t nReassGood;
  uint64_t nReassBad;
} RxL2Counters;

typedef struct RxL3Counters
{
  uint64_t nInterests;
  uint64_t nData;
  uint64_t nNacks;
} RxL3Counters;

typedef struct TxL2Counters
{
  uint64_t nFrames;
  uint64_t nOctets;

  uint64_t nFragGood;
  uint64_t nFragBad;
} TxL2Counters;

typedef struct TxL3Counters
{
  uint64_t nInterests;
  uint64_t nData;
  uint64_t nNacks;
} TxL3Counters;

typedef struct FaceCounters
{
  RxL2Counters rxl2;
  RxL3Counters rxl3;
  TxL2Counters txl2;
  TxL3Counters txl3;
} FaceCounters;

#endif // NDN_DPDK_IFACE_COUNTERS_H
