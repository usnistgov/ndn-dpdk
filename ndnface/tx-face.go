package ndnface

/*
#include "tx-face.h"
*/
import "C"
import (
	"fmt"
	"unsafe"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/ndn"
)

func SizeofHeaderMempoolDataRoom() uint16 {
	return uint16(C.TxFace_GetHeaderMempoolDataRoom())
}

type TxFace struct {
	c *C.TxFace
}

func NewTxFace(q dpdk.EthTxQueue, indirectMp dpdk.PktmbufPool,
	headerMp dpdk.PktmbufPool) (face TxFace, e error) {
	face.c = (*C.TxFace)(C.calloc(1, C.sizeof_TxFace))
	face.c.port = C.uint16_t(q.GetPort())
	face.c.queue = C.uint16_t(q.GetQueue())
	face.c.indirectMp = (*C.struct_rte_mempool)(indirectMp.GetPtr())
	face.c.headerMp = (*C.struct_rte_mempool)(headerMp.GetPtr())

	ok := C.TxFace_Init(face.c)
	if !ok {
		return face, dpdk.GetErrno()
	}
	return face, nil
}

func (face TxFace) Close() {
	C.TxFace_Close(face.c)
	C.free(unsafe.Pointer(face.c))
}

func (face TxFace) TxBurst(pkts []ndn.Packet) {
	if len(pkts) == 0 {
		return
	}
	C.TxFace_TxBurst(face.c, (**C.struct_rte_mbuf)(unsafe.Pointer(&pkts[0])), C.uint16_t(len(pkts)))
}

type TxFaceCounters struct {
	NInterests uint64
	NData      uint64
	NNacks     uint64

	NFrames uint64 // total L2 frames
	NOctets uint64

	NAllocFails    uint64
	NBursts        uint64
	NZeroBursts    uint64
	NPartialBursts uint64
}

func (face TxFace) GetCounters() (cnt TxFaceCounters) {
	cnt.NInterests = uint64(face.c.nPkts[ndn.NdnPktType_Interest])
	cnt.NData = uint64(face.c.nPkts[ndn.NdnPktType_Interest])
	cnt.NNacks = uint64(face.c.nPkts[ndn.NdnPktType_Interest])

	cnt.NFrames = uint64(face.c.nPkts[ndn.NdnPktType_None]) + cnt.NInterests + cnt.NData + cnt.NNacks
	cnt.NOctets = uint64(face.c.nOctets)

	cnt.NAllocFails = uint64(face.c.nAllocFails)
	cnt.NBursts = uint64(face.c.nBursts)
	cnt.NZeroBursts = uint64(face.c.nZeroBursts)
	cnt.NPartialBursts = uint64(face.c.nPartialBursts)

	return cnt
}

func (cnt TxFaceCounters) String() string {
	return fmt.Sprintf(
		"L3 %dI %dD %dN, L2 %dfrm %db; %d alloc-fail, %d bursts, %d partial, %d zero",
		cnt.NInterests, cnt.NData, cnt.NNacks, cnt.NFrames, cnt.NOctets,
		cnt.NAllocFails, cnt.NBursts, cnt.NPartialBursts, cnt.NZeroBursts)
}
