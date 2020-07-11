#ifndef NDN_DPDK_CONTAINER_MINTMR_MINTMRTEST_MINTMRTEST_H
#define NDN_DPDK_CONTAINER_MINTMR_MINTMRTEST_MINTMRTEST_H

#include "../../csrc/mintmr/mintmr.h"

typedef struct MinTmrTestRecord
{
  MinTmr tmr;
  int n;
} MinTmrTestRecord;

MinSched*
MinTmrTest_MakeSched(int nSlotBits, TscDuration interval);

MinTmrTestRecord*
MinTmrTest_NewRecord(int n);

#endif // NDN_DPDK_CONTAINER_MINTMR_MINTMRTEST_MINTMRTEST_H
