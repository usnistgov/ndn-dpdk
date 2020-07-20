package mintmrtest

/*
#include "mintmrtest.h"
extern void go_TriggerRecord(MinTmr* tmr, void* arg);
*/
import "C"
import (
	"testing"
	"time"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

func ctestMinTmr(t *testing.T) {
	assert, _ := testenv.MakeAR(t)
	C.c_ClearRecords()
	schedArg = C.malloc(1)
	defer C.free(schedArg)

	// 2^5 slots * 100ms = 3200ms
	sched := C.MinSched_New(5, C.TscDuration(eal.ToTscDuration(100*time.Millisecond)),
		C.MinTmrCallback(C.go_TriggerRecord), schedArg)
	defer C.MinSched_Close(sched)

	setTimer := func(i int, after time.Duration) bool {
		return bool(C.MinTmr_After(&C.records[i].tmr, C.TscDuration(eal.ToTscDuration(after)), sched))
	}

	checkRecords := func(t1, t2, t3, t4, t5 int) {
		assert.EqualValues(t1, C.records[1].triggered)
		assert.EqualValues(t2, C.records[2].triggered)
		assert.EqualValues(t3, C.records[3].triggered)
		assert.EqualValues(t4, C.records[4].triggered)
		assert.EqualValues(t5, C.records[5].triggered)
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

var schedArg unsafe.Pointer

//export go_TriggerRecord
func go_TriggerRecord(tmr *C.MinTmr, arg unsafe.Pointer) {
	if arg != schedArg {
		panic(arg)
	}

	rec := (*C.MinTmrTestRecord)(unsafe.Pointer(tmr))
	rec.triggered++
}
