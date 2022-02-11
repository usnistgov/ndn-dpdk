package coretest

/*
#include "../../csrc/core/mintmr.h"

typedef struct MinTmrTestRecord
{
  MinTmr tmr;
  int triggered;
} MinTmrTestRecord;

extern void go_TriggerRecord(MinTmr* tmr, uintptr_t arg);
*/
import "C"
import (
	"testing"
	"time"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

const schedCtx = 0xBBB89535DC3634F7

func ctestMinTmr(t *testing.T) {
	assert, _ := testenv.MakeAR(t)
	records := (*[6]C.MinTmrTestRecord)(C.calloc(6, C.size_t(unsafe.Sizeof(C.MinTmrTestRecord{}))))
	defer C.free(unsafe.Pointer(records))

	// 2^5 slots * 100ms = 3200ms
	sched := C.MinSched_New(5, C.TscDuration(eal.ToTscDuration(100*time.Millisecond)), C.MinTmrCb(C.go_TriggerRecord), schedCtx)
	defer C.MinSched_Close(sched)

	setTimer := func(i int, after time.Duration) bool {
		return bool(C.MinTmr_After(&records[i].tmr, C.TscDuration(eal.ToTscDuration(after)), sched))
	}

	checkRecords := func(t1, t2, t3, t4, t5 int) {
		assert.EqualValues(t1, records[1].triggered)
		assert.EqualValues(t2, records[2].triggered)
		assert.EqualValues(t3, records[3].triggered)
		assert.EqualValues(t4, records[4].triggered)
		assert.EqualValues(t5, records[5].triggered)
	}

	assert.False(setTimer(1, 3300*time.Millisecond)) // tmr1 is too far into the future
	assert.True(setTimer(2, 500*time.Millisecond))   // tmr2 will expire at 500
	checkRecords(0, 0, 0, 0, 0)

	time.Sleep(200 * time.Millisecond) // now is 200
	C.MinSched_Trigger(sched)
	checkRecords(0, 0, 0, 0, 0)
	assert.True(setTimer(3, 500*time.Millisecond))  // evt3 will expire at 700
	assert.True(setTimer(4, 2600*time.Millisecond)) // evt4 will expire at 2800
	assert.True(setTimer(5, 510*time.Millisecond))  // evt3 will expire at 710

	time.Sleep(700 * time.Millisecond) // now is 900
	C.MinSched_Trigger(sched)
	checkRecords(0, 1, 1, 0, 1)

	time.Sleep(1500 * time.Millisecond) // now is 2400
	C.MinSched_Trigger(sched)
	checkRecords(0, 1, 1, 0, 1)

	time.Sleep(600 * time.Millisecond) // now is 3000
	C.MinSched_Trigger(sched)
	checkRecords(0, 1, 1, 1, 1)
}

//export go_TriggerRecord
func go_TriggerRecord(tmr *C.MinTmr, arg C.uintptr_t) {
	if arg != schedCtx {
		panic(arg)
	}

	rec := (*C.MinTmrTestRecord)(unsafe.Pointer(tmr))
	rec.triggered++
}
