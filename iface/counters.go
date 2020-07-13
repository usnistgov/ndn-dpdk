package iface

/*
#include "../csrc/iface/face.h"
*/
import "C"
import (
	"fmt"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/runningstat"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

// Counters contains basic face counters.
type Counters struct {
	RxFrames uint64 // RX total frames
	RxOctets uint64 // RX total bytes

	DecodeErrs uint64 // decode errors
	Reass      InOrderReassemblerCounters

	RxInterests uint64 // RX Interest packets
	RxData      uint64 // RX Data packets
	RxNacks     uint64 // RX Nack packets

	InterestLatency runningstat.Snapshot
	DataLatency     runningstat.Snapshot
	NackLatency     runningstat.Snapshot

	TxInterests uint64 // TX Interest packets
	TxData      uint64 // TX Data packets
	TxNacks     uint64 // TX Nack packets

	FragGood    uint64 // fragmentated L3 packets
	FragBad     uint64 // fragmentation failures
	TxAllocErrs uint64 // allocation errors during TX
	TxDropped   uint64 // L2 frames dropped due to full queue
	TxFrames    uint64 // sent total frames
	TxOctets    uint64 // sent total bytes
}

func (cnt Counters) String() string {
	return fmt.Sprintf("RX %dfrm %db %dI %dD %dN reass=(%v) %derr TX %dfrm %db %dI %dD %dN frag=(%dgood %dbad) alloc=%derr %ddropped",
		cnt.RxFrames, cnt.RxOctets, cnt.RxInterests, cnt.RxData, cnt.RxNacks, cnt.Reass, cnt.DecodeErrs,
		cnt.TxFrames, cnt.TxOctets, cnt.TxInterests, cnt.TxData, cnt.TxNacks, cnt.FragGood, cnt.FragBad, cnt.TxAllocErrs, cnt.TxDropped)
}

// ReadCounters retrieves basic face counters.
func (f *face) ReadCounters() (cnt Counters) {
	c := f.ptr()
	if c.impl == nil {
		return cnt
	}

	rxC := &c.impl.rx
	cnt.Reass = InOrderReassemblerFromPtr(unsafe.Pointer(&rxC.reassembler)).ReadCounters()
	for i := 0; i < C.RXPROC_MAX_THREADS; i++ {
		rxtC := &rxC.threads[i]
		cnt.RxFrames += uint64(rxtC.nFrames[ndni.PktFragment])
		cnt.RxOctets += uint64(rxtC.nOctets)
		cnt.DecodeErrs += uint64(rxtC.nDecodeErr)
		cnt.RxInterests += uint64(rxtC.nFrames[ndni.PktInterest])
		cnt.RxData += uint64(rxtC.nFrames[ndni.PktData])
		cnt.RxNacks += uint64(rxtC.nFrames[ndni.PktNack])
	}
	cnt.RxFrames += cnt.RxInterests + cnt.RxData + cnt.RxNacks

	txC := &c.impl.tx

	readLatencyStat := func(c *C.RunningStat) runningstat.Snapshot {
		return runningstat.FromPtr(unsafe.Pointer(c)).Read().Scale(eal.GetNanosInTscUnit())
	}
	cnt.InterestLatency = readLatencyStat(&txC.latency[ndni.PktInterest])
	cnt.DataLatency = readLatencyStat(&txC.latency[ndni.PktData])
	cnt.NackLatency = readLatencyStat(&txC.latency[ndni.PktNack])
	cnt.TxInterests = cnt.InterestLatency.Count()
	cnt.TxData = cnt.DataLatency.Count()
	cnt.TxNacks = cnt.NackLatency.Count()

	cnt.FragGood = uint64(txC.nL3Fragmented)
	cnt.FragBad = uint64(txC.nL3OverLength + txC.nAllocFails)
	cnt.TxAllocErrs = uint64(txC.nAllocFails)
	cnt.TxDropped = uint64(txC.nDroppedFrames)
	cnt.TxFrames = uint64(txC.nFrames - txC.nDroppedFrames)
	cnt.TxOctets = uint64(txC.nOctets - txC.nDroppedOctets)

	return cnt
}
