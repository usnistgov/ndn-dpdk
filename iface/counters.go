package iface

/*
#include "counters.h"
*/
import "C"
import (
	"fmt"
	"unsafe"
)

func checkCountersDef() struct{} {
	var cnt Counters
	if unsafe.Sizeof(cnt) != C.sizeof_FaceCounters {
		panic("iface.FaceCounters definition does not match C.FaceCounters")
	}
	return struct{}{}
}

var checkCountersDefOk = checkCountersDef()

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

type Counters struct {
	RxL2 RxL2Counters
	RxL3 RxL3Counters
	TxL2 TxL2Counters
	TxL3 TxL3Counters
}

func (cnt Counters) String() string {
	return fmt.Sprintf("RX %v %v; TX %v %v", cnt.RxL2, cnt.RxL3, cnt.TxL2, cnt.TxL3)
}
