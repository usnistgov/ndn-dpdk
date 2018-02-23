#include "test-mintmr.h"
#include "_cgo_export.h"

MinSched*
MinTmrTest_MakeSched(int nSlotBits, TscDuration interval)
{
  return MinSched_New(nSlotBits, interval, MinTmrTest_TriggerRecord);
}

MinTmrTestRecord*
MinTmrTest_NewRecord(int n)
{
  MinTmrTestRecord* rec = malloc(sizeof(MinTmrTestRecord));
  MinTmr_Init(&rec->tmr);
  rec->n = n;
  return rec;
}

void
MinTmrTest_TriggerRecord(MinTmr* tmr)
{
  MinTmrTestRecord* rec = container_of(tmr, MinTmrTestRecord, tmr);
  go_TriggerRecord(rec->n);
  free(rec);
}
