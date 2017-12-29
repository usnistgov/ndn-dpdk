package ethface

/*
#include "eth-face.h"
*/
import "C"
import "fmt"

type RxCounters struct {
	NInterests uint64
	NData      uint64
	NNacks     uint64

	NFrames uint64 // total L2 frames
	NOctets uint64
}

func (face EthFace) GetRxCounters() (cnt RxCounters) {
	faceC := face.getPtr()

	cnt.NInterests = uint64(faceC.rx.nInterestPkts)
	cnt.NData = uint64(faceC.rx.nDataPkts)

	cnt.NFrames = uint64(faceC.rx.nFrames)

	return cnt
}

func (cnt RxCounters) String() string {
	return fmt.Sprintf(
		"L3 %dI %dD %dN, L2 %dfrm %db",
		cnt.NInterests, cnt.NData, 0, cnt.NFrames, 0)
}
