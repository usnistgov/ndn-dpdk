#ifndef NDNDPDK_CONTAINER_MINTMR_MINTMR_TEST_H
#define NDNDPDK_CONTAINER_MINTMR_MINTMR_TEST_H

#include "../../../csrc/mintmr/mintmr.h"

typedef struct MinTmrTestRecord
{
  MinTmr tmr;
  int n;
} MinTmrTestRecord;

MinSched*
MinTmrTest_MakeSched(int nSlotBits, TscDuration interval);

MinTmrTestRecord*
MinTmrTest_NewRecord(int n);

#endif // NDNDPDK_CONTAINER_MINTMR_MINTMR_TEST_H
