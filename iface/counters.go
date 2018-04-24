package iface

/*
#include "face.h"
*/
import "C"
import (
	"fmt"

	"ndn-dpdk/ndn"
)

type RxL2Counters struct {
	NFrames uint64 // total frames
	NOctets uint64 // total bytes

	NReassGood uint64 // reassembled L3 packets
	NReassBad  uint64 // reassembly failures (discarding reassembly queue)
}

func (cnt RxL2Counters) String() string {
	return fmt.Sprintf("%dfrm %db reass=(%dgood %dbad)", cnt.NFrames, cnt.NOctets,
		cnt.NReassGood, cnt.NReassBad)
}

type RxL3Counters struct {
	NInterests uint64
	NData      uint64
	NNacks     uint64
}

func (cnt RxL3Counters) String() string {
	return fmt.Sprintf("%dI %dD %dN", cnt.NInterests, cnt.NData, cnt.NNacks)
}

type TxL2Counters struct {
	NFrames uint64 // total frames
	NOctets uint64 // total bytes

	NFragGood uint64 // fragmentated L3 packets
	NFragBad  uint64 // fragmentation failures
}

func (cnt TxL2Counters) String() string {
	return fmt.Sprintf("%dfrm %db frag=(%dgood %dbad)", cnt.NFrames, cnt.NOctets,
		cnt.NFragGood, cnt.NFragBad)
}

type TxL3Counters struct {
	NInterests uint64
	NData      uint64
	NNacks     uint64
}

func (cnt TxL3Counters) String() string {
	return fmt.Sprintf("%dI %dD %dN", cnt.NInterests, cnt.NData, cnt.NNacks)
}

// Basic face counters.
type Counters struct {
	RxL2 RxL2Counters
	RxL3 RxL3Counters
	TxL2 TxL2Counters
	TxL3 TxL3Counters
}

func (cnt Counters) String() string {
	return fmt.Sprintf("RX %s %s; TX %s %s", cnt.RxL2, cnt.RxL3, cnt.TxL2, cnt.TxL3)
}

func (cnt *Counters) readFrom(faceC *C.Face) {
	if faceC == nil || faceC.impl == nil {
		return
	}

	rxC := &faceC.impl.rx
	txC := &faceC.impl.tx

	cnt.RxL2.NFrames = uint64(rxC.nFrames[ndn.L3PktType_None])
	cnt.RxL2.NOctets = uint64(rxC.nOctets)
	cnt.RxL2.NReassGood = uint64(rxC.reassembler.nDelivered)
	cnt.RxL2.NReassBad = uint64(rxC.reassembler.nIncomplete)

	cnt.RxL3.NInterests = uint64(rxC.nFrames[ndn.L3PktType_Interest])
	cnt.RxL3.NData = uint64(rxC.nFrames[ndn.L3PktType_Data])
	cnt.RxL3.NNacks = uint64(rxC.nFrames[ndn.L3PktType_Nack])

	cnt.TxL2.NFrames = uint64(txC.nFrames[ndn.L3PktType_None])
	cnt.TxL2.NOctets = uint64(txC.nOctets)
	cnt.TxL2.NFragGood = uint64(txC.nL3Fragmented)
	cnt.TxL2.NFragBad = uint64(txC.nL3OverLength + txC.nAllocFails)

	cnt.TxL3.NInterests = uint64(txC.nFrames[ndn.L3PktType_Interest])
	cnt.TxL3.NData = uint64(txC.nFrames[ndn.L3PktType_Data])
	cnt.TxL3.NNacks = uint64(txC.nFrames[ndn.L3PktType_Nack])
	cnt.TxL2.NFrames += cnt.TxL3.NInterests + cnt.TxL3.NData + cnt.TxL3.NNacks
}
