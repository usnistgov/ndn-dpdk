package ndnface

/*
#include "tx-face.h"
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/ndn"
)

type TxFace struct {
	c *C.TxFace
}

func NewTxFace(q dpdk.EthTxQueue) (face TxFace, e error) {
	face.c = (*C.TxFace)(C.calloc(1, C.sizeof_TxFace))
	face.c.port = C.uint16_t(q.GetPort())
	face.c.queue = C.uint16_t(q.GetQueue())
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

func (face TxFace) TxBurst(pkts []ndn.Packet) int {
	if len(pkts) == 0 {
		return 0
	}
	res := C.TxFace_TxBurst(face.c, (**C.struct_rte_mbuf)(unsafe.Pointer(&pkts[0])),
		C.uint16_t(len(pkts)))
	return int(res)
}

type TxFaceCounters struct {
	NInterests uint64
	NData      uint64
	NNacks     uint64

	NFrames uint64
	NOctets uint64
}

func (face TxFace) GetCounters() (cnt TxFaceCounters) {
	cnt.NInterests = uint64(face.c.nPkts[ndn.NdnPktType_Interest])
	cnt.NData = uint64(face.c.nPkts[ndn.NdnPktType_Interest])
	cnt.NNacks = uint64(face.c.nPkts[ndn.NdnPktType_Interest])

	cnt.NFrames = uint64(face.c.nPkts[ndn.NdnPktType_None]) + cnt.NInterests + cnt.NData + cnt.NNacks
	cnt.NOctets = uint64(face.c.nOctets)
	return cnt
}
