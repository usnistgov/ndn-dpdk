package iface

/*
#include "face.h"
*/
import "C"
import (
	"fmt"
	"unsafe"

	"ndn-dpdk/core/running_stat"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/ndn"
)

type BaseFace struct {
	id FaceId
}

func (face BaseFace) getPtr() *C.Face {
	return &C.__gFaces[face.id]
}

// Get native *C.Face pointer to use in other packages.
func (face BaseFace) GetPtr() unsafe.Pointer {
	return unsafe.Pointer(face.getPtr())
}

// Initialize BaseFace.
// Allocate FaceImpl on specified NumaSocket.
func (face *BaseFace) InitBaseFace(id FaceId, sizeofPriv int, socket dpdk.NumaSocket) {
	face.id = id

	if socket == dpdk.NUMA_SOCKET_ANY {
		if lc := dpdk.GetCurrentLCore(); lc.IsValid() {
			socket = lc.GetNumaSocket()
		} else {
			socket = 0
		}
	}

	faceC := face.getPtr()
	*faceC = C.Face{}
	faceC.id = C.FaceId(face.id)
	faceC.state = C.FACESTA_UP
	faceC.numaSocket = C.int(socket)

	sizeofImpl := int(C.sizeof_FaceImpl) + sizeofPriv
	faceC.impl = (*C.FaceImpl)(dpdk.ZmallocAligned("FaceImpl", sizeofImpl, 1, socket))
}

// Close BaseFace.
// Deallocate FaceImpl.
func (face BaseFace) CloseBaseFace() {
	faceC := face.getPtr()
	faceC.state = C.FACESTA_REMOVED
	dpdk.Free(faceC.impl)
}

func (face BaseFace) GetFaceId() FaceId {
	return face.id
}

func (face BaseFace) GetNumaSocket() dpdk.NumaSocket {
	return dpdk.NumaSocket(face.getPtr().numaSocket)
}

// Make TxBurst thread-safe.
//
// Initially, Face_TxBurst (or Face.TxBurst in Go) is non-thread-safe.
// This function adds a software queue on the face, to make TxBurst thread safe.
// The face must then be added to a TxLooper.
//
// queueCapacity must be (2^q).
func (face BaseFace) EnableThreadSafeTx(queueCapacity int) error {
	faceC := face.getPtr()
	if faceC.threadSafeTxQueue != nil {
		return fmt.Errorf("Face %d already has thread-safe TX", face.GetFaceId())
	}

	r, e := dpdk.NewRing(fmt.Sprintf("FaceTsTx_%d", face.GetFaceId()),
		queueCapacity, face.GetNumaSocket(), false, true)
	if e != nil {
		return e
	}

	faceC.threadSafeTxQueue = (*C.struct_rte_ring)(r.GetPtr())
	return nil
}

func (face BaseFace) TxBurst(pkts []ndn.Packet) {
	if len(pkts) == 0 {
		return
	}
	C.Face_TxBurst(C.FaceId(face.id), (**C.Packet)(unsafe.Pointer(&pkts[0])), C.uint16_t(len(pkts)))
}

func (face BaseFace) ReadCounters() (cnt Counters) {
	cnt.readFrom(face.getPtr())
	return cnt
}

func (face BaseFace) ReadExCounters() interface{} {
	return nil
}

func (face BaseFace) ReadLatency() running_stat.Snapshot {
	faceC := face.getPtr()
	latencyStat := running_stat.FromPtr(unsafe.Pointer(&faceC.impl.latencyStat))
	return running_stat.TakeSnapshot(latencyStat).Multiply(dpdk.GetNanosInTscUnit())
}
