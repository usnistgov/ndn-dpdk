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

// Mempools for face construction.
type Mempools struct {
	// mempool for indirect mbufs
	IndirectMp dpdk.PktmbufPool

	// mempool for name linearize upon RX;
	// dataroom must be at least NAME_MAX_LENGTH
	NameMp dpdk.PktmbufPool

	// mempool for NDNLP headers upon TX;
	// dataroom must be at least transport-specific-headroom + PrependLpHeader_GetHeadroom()
	HeaderMp dpdk.PktmbufPool
}

type BaseFace struct {
	id FaceId
}

func (face *BaseFace) getPtr() *C.Face {
	return &C.__gFaces[face.id]
}

// Get native *C.Face pointer to use in other packages.
func (face *BaseFace) GetPtr() unsafe.Pointer {
	return unsafe.Pointer(face.getPtr())
}

// Initialize BaseFace before lower-layer initialization.
// Allocate FaceImpl on specified NumaSocket.
func (face *BaseFace) InitBaseFace(id FaceId, sizeofPriv int, socket dpdk.NumaSocket) error {
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

	return nil

}

// Initialize BaseFace after lower-layer initialization.
// mtu: transport MTU available for NDNLP packets.
// headroom: headroom before NDNLP header, as required by transport.
func (face *BaseFace) FinishInitBaseFace(txQueueCapacity, mtu, headroom int, mempools Mempools) error {
	faceC := face.getPtr()

	r, e := dpdk.NewRing(fmt.Sprintf("FaceTx_%d", face.GetFaceId()),
		txQueueCapacity, face.GetNumaSocket(), false, true)
	if e != nil {
		face.clear()
		return e
	}
	faceC.txQueue = (*C.struct_rte_ring)(r.GetPtr())

	C.RunningStat_SetSampleRate(&faceC.impl.latencyStat, 16) // collect latency once every 2^16 packets

	if res := C.TxProc_Init(&faceC.impl.tx, C.uint16_t(mtu), C.uint16_t(headroom),
		(*C.struct_rte_mempool)(mempools.IndirectMp.GetPtr()), (*C.struct_rte_mempool)(mempools.HeaderMp.GetPtr())); res != 0 {
		face.clear()
		return dpdk.Errno(res)
	}

	if res := C.RxProc_Init(&faceC.impl.rx, (*C.struct_rte_mempool)(mempools.NameMp.GetPtr())); res != 0 {
		face.clear()
		return dpdk.Errno(res)
	}

	return nil
}

func (face *BaseFace) GetFaceId() FaceId {
	return face.id
}

func (face *BaseFace) GetNumaSocket() dpdk.NumaSocket {
	return dpdk.NumaSocket(face.getPtr().numaSocket)
}

func (face *BaseFace) IsClosed() bool {
	return face == nil || face.getPtr().id == C.FACEID_INVALID
}

// Prepare to close face.
func (face *BaseFace) BeforeClose() {
	id := face.GetFaceId()
	faceC := face.getPtr()
	faceC.state = C.FACESTA_DOWN
	emitter.EmitSync(evt_FaceClosing, id)
}

func (face *BaseFace) clear() {
	id := face.GetFaceId()
	faceC := face.getPtr()
	faceC.state = C.FACESTA_REMOVED
	if faceC.impl != nil {
		dpdk.Free(faceC.impl)
	}
	if faceC.txQueue != nil {
		dpdk.RingFromPtr(unsafe.Pointer(faceC.txQueue)).Close()
	}
	faceC.id = C.FACEID_INVALID
	gFaces[id] = nil
}

// Close BaseFace.
// Deallocate FaceImpl.
func (face *BaseFace) CloseBaseFace() {
	id := face.GetFaceId()
	face.clear()
	emitter.EmitSync(evt_FaceClosed, id)
}

func (face *BaseFace) IsDown() bool {
	return face.getPtr().state != C.FACESTA_UP
}

func (face *BaseFace) SetDown(isDown bool) {
	if face.IsDown() == isDown {
		return
	}
	id := face.GetFaceId()
	faceC := face.getPtr()
	if isDown {
		faceC.state = C.FACESTA_DOWN
		emitter.EmitSync(evt_FaceDown, id)
	} else {
		faceC.state = C.FACESTA_UP
		emitter.EmitSync(evt_FaceUp, id)
	}
}

func (face *BaseFace) TxBurst(pkts []ndn.Packet) {
	if len(pkts) == 0 {
		return
	}
	C.Face_TxBurst(C.FaceId(face.id), (**C.Packet)(unsafe.Pointer(&pkts[0])), C.uint16_t(len(pkts)))
}

func (face *BaseFace) ReadLatency() running_stat.Snapshot {
	faceC := face.getPtr()
	latencyStat := running_stat.FromPtr(unsafe.Pointer(&faceC.impl.latencyStat))
	return running_stat.TakeSnapshot(latencyStat).Multiply(dpdk.GetNanosInTscUnit())
}
