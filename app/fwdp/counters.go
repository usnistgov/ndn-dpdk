package fwdp

/*
#include "input.h"
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/core/running_stat"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
)

// Dataplane counters.
type Counters struct {
	NInputs int // count of inputs
	NFwds   int // count of fwds
	Inputs  []InputCounters
	Fwds    []FwdCounters
}

// Information and counters about an input process.
type InputCounters struct {
	LCore dpdk.LCore
	Faces []iface.FaceId
}

// Information and counters about a forwarding process.
type FwdCounters struct {
	LCore dpdk.LCore

	QueueCapacity int                   // input queue capacity
	NQueueDrops   uint64                // count of packets dropped because input queue is full
	TimeSinceRx   running_stat.Snapshot // input latency in nanos

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
		ic.Faces = dp.inputRxLoopers[i].ListFacesInRxLoop()
	}

	for i, fwd := range dp.fwds {
		fc := &cnt.Fwds[i]
		fc.LCore = dp.fwdLCores[i]

		fwdQ := dpdk.RingFromPtr(unsafe.Pointer(fwd.queue))
		fc.QueueCapacity = fwdQ.GetCapacity()

		timeSinceRxStat := running_stat.FromPtr(unsafe.Pointer(&fwd.timeSinceRxStat))
		fc.TimeSinceRx = running_stat.TakeSnapshot(timeSinceRxStat).Multiply(dpdk.GetNanosInTscUnit())

		fc.HeaderMpUsage = dpdk.MempoolFromPtr(unsafe.Pointer(fwd.headerMp)).CountInUse()
		fc.IndirectMpUsage = dpdk.MempoolFromPtr(unsafe.Pointer(fwd.indirectMp)).CountInUse()

		for _, input := range dp.inputs {
			inputConn := C.FwInput_GetConn(input, C.uint8_t(i))
			fc.NQueueDrops += uint64(inputConn.nDrops)
		}
	}

	return cnt
}
