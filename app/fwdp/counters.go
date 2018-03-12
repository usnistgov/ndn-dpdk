package fwdp

/*
#include "input.h"
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/dpdk"
)

// Dataplane's detailed counters.
type Counters struct {
	NInputs int // count of inputs
	NFwds   int // count of fwds
	Inputs  []InputCounters
	Fwds    []FwdCounters
}

// Information and counters about an input process.
type InputCounters struct {
	LCore dpdk.LCore
}

// Information and counters about a forwarding process.
type FwdCounters struct {
	LCore dpdk.LCore

	QueueCapacity int    // input queue capacity
	NQueueDrops   uint64 // count of packets dropped because input queue is full

	HeaderMpUsage   int
	IndirectMpUsage int
}

// Read counters about input and forwarding processes.
func (dp *DataPlane) ReadCounters() (cnt Counters) {
	cnt.NInputs = len(dp.inputs)
	cnt.Inputs = make([]InputCounters, cnt.NInputs)
	cnt.NFwds = len(dp.fwds)
	cnt.Fwds = make([]FwdCounters, cnt.NFwds)

	for i := range dp.inputs {
		ic := &cnt.Inputs[i]
		ic.LCore = dp.inputLCores[i]
	}

	for i, fwd := range dp.fwds {
		fc := &cnt.Fwds[i]
		fc.LCore = dp.fwdLCores[i]
		fc.QueueCapacity = dpdk.RingFromPtr(unsafe.Pointer(fwd.queue)).GetCapacity()
		fc.HeaderMpUsage = dpdk.MempoolFromPtr(unsafe.Pointer(fwd.headerMp)).CountInUse()
		fc.IndirectMpUsage = dpdk.MempoolFromPtr(unsafe.Pointer(fwd.indirectMp)).CountInUse()
		for _, input := range dp.inputs {
			inputConn := C.FwInput_GetConn(input, C.uint8_t(i))
			fc.NQueueDrops += uint64(inputConn.nDrops)
		}
	}

	return cnt
}
