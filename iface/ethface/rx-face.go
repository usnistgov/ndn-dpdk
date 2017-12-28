package ethface

/*
#include "rx-face.h"
*/
import "C"
import (
	"fmt"
	"unsafe"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

type RxFace struct {
	c *C.EthRxFace
}

func NewRxFace(q dpdk.EthRxQueue) (face RxFace, e error) {
	if !hasValidFaceId(q.GetPort()) {
		return face, fmt.Errorf("port number is too large")
	}

	face.c = (*C.EthRxFace)(C.calloc(1, C.sizeof_EthRxFace))
	face.c.port = C.uint16_t(q.GetPort())
	face.c.queue = C.uint16_t(q.GetQueue())
	return face, nil
}

func (face RxFace) GetFaceId() iface.FaceId {
	return FaceIdFromEthDev(uint16(face.c.port))
}

func (face RxFace) Close() error {
	C.free(unsafe.Pointer(face.c))
	return nil
}

func (face RxFace) RxBurst(pkts []ndn.Packet) int {
	if len(pkts) == 0 {
		return 0
	}
	res := C.EthRxFace_RxBurst(face.c, (**C.struct_rte_mbuf)(unsafe.Pointer(&pkts[0])),
		C.uint16_t(len(pkts)))
	return int(res)
}

type RxFaceCounters struct {
	NInterests uint64
	NData      uint64
	NNacks     uint64

	NFrames uint64 // total L2 frames
	NOctets uint64
}

func (face RxFace) GetCounters() (cnt RxFaceCounters) {
	cnt.NInterests = uint64(face.c.nInterestPkts)
	cnt.NData = uint64(face.c.nDataPkts)

	cnt.NFrames = uint64(face.c.nFrames)

	return cnt
}

func (cnt RxFaceCounters) String() string {
	return fmt.Sprintf(
		"L3 %dI %dD %dN, L2 %dfrm %db",
		cnt.NInterests, cnt.NData, 0, cnt.NFrames, 0)
}
