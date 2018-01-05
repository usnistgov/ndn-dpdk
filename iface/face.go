package iface

/*
#include "face.h"
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/ndn"
)

type Face struct {
	c *C.Face
}

// Construct Face from native *C.Face pointer.
func FaceFromPtr(ptr unsafe.Pointer) (face Face) {
	face.c = (*C.Face)(ptr)
	return face
}

// Get native *C.Face pointer to use in other packages.
func (face Face) GetPtr() unsafe.Pointer {
	return unsafe.Pointer(face.c)
}

func (face Face) GetFaceId() FaceId {
	return FaceId(face.c.id)
}

func (face Face) GetNumaSocket() dpdk.NumaSocket {
	return dpdk.NumaSocket(C.Face_GetNumaSocket(face.c))
}

func (face Face) Close() error {
	ok := C.Face_Close(face.c)
	C.free(unsafe.Pointer(face.c))

	if !ok {
		return dpdk.GetErrno()
	}
	return nil
}

func (face Face) RxBurst(pkts []ndn.Packet) int {
	if len(pkts) == 0 {
		return 0
	}
	res := C.Face_RxBurst(face.c, (**C.struct_rte_mbuf)(unsafe.Pointer(&pkts[0])), C.uint16_t(len(pkts)))
	return int(res)
}

func (face Face) TxBurst(pkts []ndn.Packet) {
	if len(pkts) == 0 {
		return
	}
	C.Face_TxBurst(face.c, (**C.struct_rte_mbuf)(unsafe.Pointer(&pkts[0])), C.uint16_t(len(pkts)))
}

func (face Face) ReadCounters() Counters {
	var cnt Counters
	C.Face_ReadCounters(face.c, (*C.FaceCounters)(unsafe.Pointer(&cnt)))
	return cnt
}
