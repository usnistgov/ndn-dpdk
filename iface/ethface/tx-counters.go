package ethface

/*
#include "eth-face.h"
*/
import "C"
import (
	"fmt"

	"ndn-dpdk/ndn"
)

type TxCounters struct {
	NInterests uint64
	NData      uint64
	NNacks     uint64

	NFrames uint64 // total L2 frames
	NOctets uint64

	NL3Bursts     uint64
	NL3OverLength uint64
	NAllocFails   uint64
	NL2Bursts     uint64
	NL2Incomplete uint64
}

func (face EthFace) GetTxCounters() (cnt TxCounters) {
	faceC := face.getPtr()

	cnt.NInterests = uint64(faceC.tx.nPkts[ndn.NdnPktType_Interest])
	cnt.NData = uint64(faceC.tx.nPkts[ndn.NdnPktType_Data])
	cnt.NNacks = uint64(faceC.tx.nPkts[ndn.NdnPktType_Nack])

	cnt.NFrames = uint64(faceC.tx.nPkts[ndn.NdnPktType_None]) + cnt.NInterests + cnt.NData + cnt.NNacks
	cnt.NOctets = uint64(faceC.tx.nOctets)

	cnt.NL3Bursts = uint64(faceC.tx.nL3Bursts)
	cnt.NL3OverLength = uint64(faceC.tx.nL3OverLength)
	cnt.NAllocFails = uint64(faceC.tx.nAllocFails)
	cnt.NL2Bursts = uint64(faceC.tx.nL2Bursts)
	cnt.NL2Incomplete = uint64(faceC.tx.nL2Incomplete)

	return cnt
}

func (cnt TxCounters) String() string {
	return fmt.Sprintf(
		"%dI %dD %dN %dfrm %db; L3 %dbursts %doverlen %dallocfail; L2 %dbursts, %dincomplete",
		cnt.NInterests, cnt.NData, cnt.NNacks, cnt.NFrames, cnt.NOctets,
		cnt.NL3Bursts, cnt.NL3OverLength, cnt.NAllocFails, cnt.NL2Bursts, cnt.NL2Incomplete)
}
