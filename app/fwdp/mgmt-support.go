package fwdp

/*
#include "fwd.h"
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/app/inputdemux"
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
}

// Read information about i-th input.
func (dp *DataPlane) ReadInputInfo(i int) (info *InputInfo) {
	if i < 0 || i >= len(dp.inputs) {
		return nil
	}
	input := dp.inputs[i]

	info = new(InputInfo)
	info.LCore = dpdk.LCORE_INVALID
	if input.rxl != nil {
		info.LCore = input.rxl.GetLCore()
		info.Faces = input.rxl.ListFaces()
	}

	return info
}

// Information and counters about a fwd process.
type FwdInfo struct {
	LCore dpdk.LCore // LCore executing this fwd process

	InputInterest FwdInputCounter
	InputData     FwdInputCounter
	InputNack     FwdInputCounter
	InputLatency  running_stat.Snapshot // input latency in nanos

	NNoFibMatch   uint64 // Interests dropped due to no FIB match
	NDupNonce     uint64 // Interests dropped due duplicate nonce
	NSgNoFwd      uint64 // Interests not forwarded by strategy
	NNackMismatch uint64 // Nack dropped due to outdated nonce

	HeaderMpUsage   int // how many entries are used in header mempool
	IndirectMpUsage int // how many entries are used in indirect mempool
}

type FwdInputCounter struct {
	NDropped   uint64 // dropped due to full queue
	NQueued    uint64 // queued
	NCongMarks uint64 // inserted congestion marks
}

func (cnt *FwdInputCounter) add(m inputdemux.DestCounters) {
	cnt.NDropped += m.NDropped
	cnt.NQueued += m.NQueued
}

// Read information about i-th fwd.
func (dp *DataPlane) ReadFwdInfo(i int) (info *FwdInfo) {
	if i < 0 || i >= len(dp.fwds) {
		return nil
	}

	info = new(FwdInfo)
	fwd := dp.fwds[i]
	info.LCore = fwd.GetLCore()

	latencyStat := running_stat.FromPtr(unsafe.Pointer(&fwd.c.latencyStat))
	info.InputLatency = running_stat.TakeSnapshot(latencyStat).Multiply(dpdk.GetNanosInTscUnit())

	info.NNoFibMatch = uint64(fwd.c.nNoFibMatch)
	info.NDupNonce = uint64(fwd.c.nDupNonce)
	info.NSgNoFwd = uint64(fwd.c.nSgNoFwd)
	info.NNackMismatch = uint64(fwd.c.nNackMismatch)

	info.HeaderMpUsage = dpdk.MempoolFromPtr(unsafe.Pointer(fwd.c.headerMp)).CountInUse()
	info.IndirectMpUsage = dpdk.MempoolFromPtr(unsafe.Pointer(fwd.c.indirectMp)).CountInUse()

	for _, input := range dp.inputs {
		info.InputInterest.add(input.demux3.GetInterestDemux().ReadDestCounters(i))
		info.InputData.add(input.demux3.GetDataDemux().ReadDestCounters(i))
		info.InputNack.add(input.demux3.GetNackDemux().ReadDestCounters(i))
	}
	info.InputInterest.NCongMarks = uint64(fwd.c.inInterestQueue.nDrops)
	info.InputData.NCongMarks = uint64(fwd.c.inDataQueue.nDrops)
	info.InputNack.NCongMarks = uint64(fwd.c.inNackQueue.nDrops)

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
	pcct := pcct.PcctFromPtr(unsafe.Pointer(*C.FwFwd_GetPcctPtr_(dp.fwds[i].c)))
	return &pcct
}
