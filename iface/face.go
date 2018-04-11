package iface

/*
#include "iface.h"
*/
import "C"
import (
	"fmt"
	"unsafe"

	"ndn-dpdk/core/running_stat"
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

// Allocate Face.c on specified NumaSocket.
// This should be only be called by subtype constructor.
// size: C.sizeof_SubTypeOfFace
func (face *Face) AllocCFace(size interface{}, socket dpdk.NumaSocket) {
	face.c = (*C.Face)(dpdk.ZmallocAligned("Face", size, 1, socket))
}

func (face Face) getPtr() *C.Face {
	return face.c
}

// Get native *C.Face pointer to use in other packages.
func (face Face) GetPtr() unsafe.Pointer {
	return unsafe.Pointer(face.c)
}

func (face Face) IsValid() bool {
	return face.c != nil
}

func (face Face) GetFaceId() FaceId {
	return FaceId(face.c.id)
}

func (face Face) GetNumaSocket() dpdk.NumaSocket {
	return dpdk.NumaSocket(C.Face_GetNumaSocket(face.c.id))
}

func (face Face) Close() error {
	ok := C.Face_Close(face.c)
	dpdk.Free(face.c)

	if !ok {
		return dpdk.GetErrno()
	}
	return nil
}

// Make TxBurst thread-safe.
//
// Initially, Face_TxBurst (or Face.TxBurst in Go) is non-thread-safe.
// This function adds a software queue on the face, to make TxBurst thread safe.
// The face must then be added to a TxLooper.
//
// queueCapacity must be (2^q).
func (face Face) EnableThreadSafeTx(queueCapacity int) error {
	if face.c.threadSafeTxQueue != nil {
		return fmt.Errorf("Face %d already has thread-safe TX", face.GetFaceId())
	}

	r, e := dpdk.NewRing(fmt.Sprintf("FaceTsTx_%d", face.GetFaceId()),
		queueCapacity, face.GetNumaSocket(), false, true)
	if e != nil {
		return e
	}

	face.c.threadSafeTxQueue = (*C.struct_rte_ring)(r.GetPtr())
	return nil
}

func (face Face) TxBurst(pkts []ndn.Packet) {
	if len(pkts) == 0 {
		return
	}
	C.__Face_TxBurst(face.c, (**C.Packet)(unsafe.Pointer(&pkts[0])), C.uint16_t(len(pkts)))
}

func (face Face) ReadCounters() Counters {
	var cnt Counters
	C.Face_ReadCounters(face.c, (*C.FaceCounters)(unsafe.Pointer(&cnt)))
	return cnt
}

// Read L3 latency statistics (in nanos).
func (face Face) ReadLatency() running_stat.Snapshot {
	latencyStat := running_stat.FromPtr(unsafe.Pointer(&face.c.latencyStat))
	return running_stat.TakeSnapshot(latencyStat).Multiply(dpdk.GetNanosInTscUnit())
}
