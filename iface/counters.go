package iface

/*
#include "face.h"
*/
import "C"
import (
	"fmt"

	"ndn-dpdk/ndn"
)

// Basic face counters.
type Counters struct {
	RxFrames uint64 // RX total frames
	RxOctets uint64 // RX total bytes

	L2DecodeErrs uint64 // L2 decode errors
	ReassBad     uint64 // reassembly failures
	ReassGood    uint64 // reassembled L3 packets

	L3DecodeErrs uint64 // L3 decode errors
	RxInterests  uint64 // RX Interest packets
	RxData       uint64 // RX Data packets
	RxNacks      uint64 // RX Nack packets

	FragGood uint64 // fragmentated L3 packets
	FragBad  uint64 // fragmentation failures

	TxAllocErrs uint64 // allocation errors during TX
	TxQueued    uint64 // L2 frames added into TX queue
	TxDropped   uint64 // L2 frames dropped because TX queue is full

	TxInterests uint64 // sent Interest packets
	TxData      uint64 // sent Data packets
	TxNacks     uint64 // sent Nack packets
	TxFrames    uint64 // sent total frames
	TxOctets    uint64 // sent total bytes
}

func (cnt Counters) String() string {
	return fmt.Sprintf("RX %dfrm %db %dI %dD %dN reass=(%dgood %dbad) l2=%derr l3=%derr TX %dfrm %db %dI %dD %dN frag=(%dgood %dbad) alloc=%derr %dqueued %ddropped",
		cnt.RxFrames, cnt.RxOctets, cnt.RxInterests, cnt.RxData, cnt.RxNacks, cnt.ReassGood, cnt.ReassBad, cnt.L2DecodeErrs, cnt.L3DecodeErrs,
		cnt.TxFrames, cnt.TxOctets, cnt.TxInterests, cnt.TxData, cnt.TxNacks, cnt.FragGood, cnt.FragBad, cnt.TxAllocErrs, cnt.TxQueued, cnt.TxDropped)
}

func (face BaseFace) ReadCounters() (cnt Counters) {
	faceC := face.getPtr()
	if faceC.impl == nil {
		return cnt
	}

	rxC := &faceC.impl.rx
	cnt.RxFrames = uint64(rxC.nFrames[ndn.L3PktType_None])
	cnt.RxOctets = uint64(rxC.nOctets)
	cnt.L2DecodeErrs = uint64(rxC.nL2DecodeErr)
	cnt.ReassGood = uint64(rxC.reassembler.nDelivered)
	cnt.ReassBad = uint64(rxC.reassembler.nIncomplete)
	cnt.L3DecodeErrs = uint64(rxC.nL3DecodeErr)
	cnt.RxInterests = uint64(rxC.nFrames[ndn.L3PktType_Interest])
	cnt.RxData = uint64(rxC.nFrames[ndn.L3PktType_Data])
	cnt.RxNacks = uint64(rxC.nFrames[ndn.L3PktType_Nack])

	txC := &faceC.impl.tx
	cnt.FragGood = uint64(txC.nL3Fragmented)
	cnt.FragBad = uint64(txC.nL3OverLength + txC.nAllocFails)
	cnt.TxAllocErrs = uint64(txC.nAllocFails)
	cnt.TxQueued = uint64(txC.nQueueAccepts)
	cnt.TxDropped = uint64(txC.nQueueRejects)
	cnt.TxInterests = uint64(txC.nFrames[ndn.L3PktType_Interest])
	cnt.TxData = uint64(txC.nFrames[ndn.L3PktType_Data])
	cnt.TxNacks = uint64(txC.nFrames[ndn.L3PktType_Nack])
	cnt.TxFrames = uint64(txC.nFrames[ndn.L3PktType_None]) + cnt.TxInterests + cnt.TxData + cnt.TxNacks
	cnt.TxOctets = uint64(txC.nOctets)

	return cnt
}

func (face BaseFace) ReadExCounters() interface{} {
	return nil
}
