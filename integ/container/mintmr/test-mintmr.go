package main

/*
#cgo CFLAGS: -m64 -pthread -O3 -g -march=native -I/usr/local/include/dpdk
#cgo LDFLAGS: -L../../../build -lndn-dpdk-mintmr -lndn-dpdk-dpdk -ldpdk
#include "test-mintmr.h"
*/
import "C"
import (
	"time"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/integ"
)

var triggered map[int]bool

func main() {
	t := new(integ.Testing)
	defer t.Close()
	assert, _ := integ.MakeAR(t)
	dpdktestenv.InitEal()
	triggered = make(map[int]bool)

	// 32 slots * 100ms = 3200ms
	sched := C.MinTmrTest_MakeSched(C.int(5),
		C.TscDuration(dpdk.ToTscDuration(100*time.Millisecond)))

	setTimer := func(n int, after time.Duration) bool {
		rec := C.MinTmrTest_NewRecord(C.int(n))
		return bool(C.MinTmr_After(&rec.tmr, C.TscDuration(dpdk.ToTscDuration(after)), sched))
	}

	assert.False(setTimer(1, 3300*time.Millisecond)) // tmr1 is too far into the future
	assert.True(setTimer(2, 500*time.Millisecond))   // tmr2 will expire at 500
	assert.Len(triggered, 0)

	time.Sleep(200 * time.Millisecond) // now is 200
	C.MinSched_Trigger(sched)
	assert.Len(triggered, 0)
	assert.True(setTimer(3, 500*time.Millisecond))  // evt3 will expire at 700
	assert.True(setTimer(4, 2600*time.Millisecond)) // evt4 will expire at 2800
	assert.True(setTimer(5, 510*time.Millisecond))  // evt3 will expire at 710

	time.Sleep(700 * time.Millisecond) // now is 900
	C.MinSched_Trigger(sched)
	assert.True(triggered[2])
	assert.True(triggered[3])
	assert.False(triggered[4])
	assert.True(triggered[5])

	time.Sleep(1500 * time.Millisecond) // now is 2400
	C.MinSched_Trigger(sched)
	assert.False(triggered[4])

	time.Sleep(600 * time.Millisecond) // now is 3000
	C.MinSched_Trigger(sched)
	assert.True(triggered[4])
}

//export go_TriggerRecord
func go_TriggerRecord(n C.int) {
	triggered[int(n)] = true
}
