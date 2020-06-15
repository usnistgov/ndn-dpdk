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
	"github.com/usnistgov/ndn-dpdk/ndn"
)

// Basic face counters.
type Counters struct {
	RxFrames uint64 // RX total frames
	RxOctets uint64 // RX total bytes

	L2DecodeErrs uint64 // L2 decode errors
	Reass        InOrderReassemblerCounters

	L3DecodeErrs uint64 // L3 decode errors
	RxInterests  uint64 // RX Interest packets
	RxData       uint64 // RX Data packets
	RxNacks      uint64 // RX Nack packets

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
	return fmt.Sprintf("RX %dfrm %db %dI %dD %dN reass=(%v) l2=%derr l3=%derr TX %dfrm %db %dI %dD %dN frag=(%dgood %dbad) alloc=%derr %ddropped",
		cnt.RxFrames, cnt.RxOctets, cnt.RxInterests, cnt.RxData, cnt.RxNacks, cnt.Reass, cnt.L2DecodeErrs, cnt.L3DecodeErrs,
		cnt.TxFrames, cnt.TxOctets, cnt.TxInterests, cnt.TxData, cnt.TxNacks, cnt.FragGood, cnt.FragBad, cnt.TxAllocErrs, cnt.TxDropped)
}

func (face FaceBase) ReadCounters() (cnt Counters) {
	faceC := face.getPtr()
	if faceC.impl == nil {
		return cnt
	}

	rxC := &faceC.impl.rx
	cnt.Reass = InOrderReassemblerFromPtr(unsafe.Pointer(&rxC.reassembler)).ReadCounters()
	for i := 0; i < C.RXPROC_MAX_THREADS; i++ {
		rxtC := &rxC.threads[i]
		cnt.RxFrames += uint64(rxtC.nFrames[ndn.L3PktType_None])
		cnt.RxOctets += uint64(rxtC.nOctets)
		cnt.L2DecodeErrs += uint64(rxtC.nL2DecodeErr)
		cnt.L3DecodeErrs += uint64(rxtC.nL3DecodeErr)
		cnt.RxInterests += uint64(rxtC.nFrames[ndn.L3PktType_Interest])
		cnt.RxData += uint64(rxtC.nFrames[ndn.L3PktType_Data])
		cnt.RxNacks += uint64(rxtC.nFrames[ndn.L3PktType_Nack])
	}

	txC := &faceC.impl.tx

	readLatencyStat := func(c *C.RunningStat) runningstat.Snapshot {
		return runningstat.FromPtr(unsafe.Pointer(c)).Read().Scale(eal.GetNanosInTscUnit())
	}
	cnt.InterestLatency = readLatencyStat(&txC.latency[ndn.L3PktType_Interest])
	cnt.DataLatency = readLatencyStat(&txC.latency[ndn.L3PktType_Data])
	cnt.NackLatency = readLatencyStat(&txC.latency[ndn.L3PktType_Nack])
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

func (face FaceBase) ReadExCounters() interface{} {
	return nil
}
