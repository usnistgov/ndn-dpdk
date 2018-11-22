package fwdp

/*
#include "input.h"
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/container/fib"
	"ndn-dpdk/container/ndt"
	"ndn-dpdk/container/pcct"
	"ndn-dpdk/core/running_stat"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
)

// Count number of input and forwarding processes.
func (dp *DataPlane) CountLCores() (nInputs int, nFwds int) {
	return len(dp.inputs), len(dp.fwds)
}

// Information and counters about an input process.
type InputInfo struct {
	LCore dpdk.LCore     // LCore executing this input process
	Faces []iface.FaceId // faces served by this input process

	NNameDisp  uint64 // packets dispatched by name
	NTokenDisp uint64 // packets dispatched by token
	NBadToken  uint64 // dropped packets due to missing or bad token
}

// Read information about i-th input.
func (dp *DataPlane) ReadInputInfo(i int) (info *InputInfo) {
	if i < 0 || i >= len(dp.inputs) {
		return nil
	}
	input := dp.inputs[i]

	info = new(InputInfo)
	info.LCore = input.lc
	if input.rxl != nil {
		info.Faces = input.rxl.ListFaces()
	}

	info.NNameDisp = uint64(input.c.nNameDisp)
	info.NTokenDisp = uint64(input.c.nTokenDisp)
	info.NBadToken = uint64(input.c.nBadToken)

	return info
}

// Information and counters about a fwd process.
type FwdInfo struct {
	LCore dpdk.LCore // LCore executing this fwd process

	QueueCapacity int                   // input queue capacity
	NQueueDrops   uint64                // packets dropped because input queue is full
	InputLatency  running_stat.Snapshot // input latency in nanos

	NNoFibMatch   uint64 // Interests dropped due to no FIB match
	NDupNonce     uint64 // Interests dropped due duplicate nonce
	NSgNoFwd      uint64 // Interests not forwarded by strategy
	NNackMismatch uint64 // Nack dropped due to outdated nonce

	HeaderMpUsage   int // how many entries are used in header mempool
	IndirectMpUsage int // how many entries are used in indirect mempool
}

// Read information about i-th fwd.
func (dp *DataPlane) ReadFwdInfo(i int) (info *FwdInfo) {
	if i < 0 || i >= len(dp.fwds) {
		return nil
	}

	info = new(FwdInfo)
	fwd := dp.fwds[i]
	info.LCore = fwd.GetLCore()

	fwdQ := dpdk.RingFromPtr(unsafe.Pointer(fwd.c.queue))
	info.QueueCapacity = fwdQ.GetCapacity()
	latencyStat := running_stat.FromPtr(unsafe.Pointer(&fwd.c.latencyStat))
	info.InputLatency = running_stat.TakeSnapshot(latencyStat).Multiply(dpdk.GetNanosInTscUnit())

	info.NNoFibMatch = uint64(fwd.c.nNoFibMatch)
	info.NDupNonce = uint64(fwd.c.nDupNonce)
	info.NSgNoFwd = uint64(fwd.c.nSgNoFwd)
	info.NNackMismatch = uint64(fwd.c.nNackMismatch)

	info.HeaderMpUsage = dpdk.MempoolFromPtr(unsafe.Pointer(fwd.c.headerMp)).CountInUse()
	info.IndirectMpUsage = dpdk.MempoolFromPtr(unsafe.Pointer(fwd.c.indirectMp)).CountInUse()

	for _, input := range dp.inputs {
		inputConn := C.FwInput_GetConn(input.c, C.uint8_t(i))
		info.NQueueDrops += uint64(inputConn.nDrops)
	}

	return info
}

// Access the NDT.
func (dp *DataPlane) GetNdt() *ndt.Ndt {
	return dp.ndt
}

// Access the FIB.
func (dp *DataPlane) GetFib() *fib.Fib {
	return dp.fib
}

// Access i-th fwd's PCCT.
func (dp *DataPlane) GetFwdPcct(i int) *pcct.Pcct {
	if i < 0 || i >= len(dp.fwds) {
		return nil
	}
	pcct := pcct.PcctFromPtr(unsafe.Pointer(*C.__FwFwd_GetPcctPtr(dp.fwds[i].c)))
	return &pcct
}
