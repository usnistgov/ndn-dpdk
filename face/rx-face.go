package face

/*
#include "rx-face.h"
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/dpdk"
)

type RxFace struct {
	c C.RxFace
}

func RxFace_GetPktPrivSize() uint16 {
	return uint16(C.RxFace_GetPktPrivSize())
}

func NewRxFace(q dpdk.EthRxQueue) RxFace {
	face := RxFace{}
	face.c.port = C.uint16_t(q.GetPort())
	face.c.queue = C.uint16_t(q.GetQueue())
	return face
}

func (face RxFace) RxBurst(pkts []Packet) int {
	if len(pkts) == 0 {
		return 0
	}
	res := C.RxFace_RxBurst(&face.c, (**C.struct_rte_mbuf)(unsafe.Pointer(&pkts[0])),
		C.uint16_t(len(pkts)))
	return int(res)
}
