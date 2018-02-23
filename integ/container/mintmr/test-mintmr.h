#ifndef NDN_DPDK_INTEG_CONTAINER_MINTMR_TEST_MINTMR_H
#define NDN_DPDK_INTEG_CONTAINER_MINTMR_TEST_MINTMR_H

/// \file

#include "../../../container/mintmr/mintmr.h"

typedef struct MinTmrTestRecord
{
  MinTmr tmr;
  int n;
} MinTmrTestRecord;

MinSched* MinTmrTest_MakeSched(int nSlotBits, TscDuration interval);

MinTmrTestRecord* MinTmrTest_NewRecord(int n);

void MinTmrTest_TriggerRecord(MinTmr* tmr);

#endif // NDN_DPDK_INTEG_CONTAINER_MINTMR_TEST_MINTMR_H
