#include "mintmr_test.h"
#include "_cgo_export.h"

void
MinTmrTest_TriggerRecord(MinTmr* tmr, void* arg)
{
  NDNDPDK_ASSERT((uintptr_t)arg == 0xEFAB9817);
  MinTmrTestRecord* rec = container_of(tmr, MinTmrTestRecord, tmr);
  go_TriggerRecord(rec->n);
  free(rec);
}

MinSched*
MinTmrTest_MakeSched(int nSlotBits, TscDuration interval)
{
  return MinSched_New(
    nSlotBits, interval, MinTmrTest_TriggerRecord, (void*)0xEFAB9817);
}

MinTmrTestRecord*
MinTmrTest_NewRecord(int n)
{
  MinTmrTestRecord* rec = malloc(sizeof(MinTmrTestRecord));
  MinTmr_Init(&rec->tmr);
  rec->n = n;
  return rec;
}
