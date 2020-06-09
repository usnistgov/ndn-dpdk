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

// Base type to implement IFace.
type FaceBase struct {
	id FaceId
}

func (face *FaceBase) getPtr() *C.Face {
	return &C.gFaces_[face.id]
}

// Get native *C.Face pointer to use in other packages.
func (face *FaceBase) GetPtr() unsafe.Pointer {
	return unsafe.Pointer(face.getPtr())
}

// Initialize FaceBase before lower-layer initialization.
// Allocate FaceImpl on specified NumaSocket.
func (face *FaceBase) InitFaceBase(id FaceId, sizeofPriv int, socket dpdk.NumaSocket) error {
	face.id = id

	if socket.IsAny() {
		if lc := dpdk.GetCurrentLCore(); lc.IsValid() {
			socket = lc.GetNumaSocket()
		} else {
			socket = dpdk.NumaSocketFromID(0) // TODO what if socket 0 is unavailable?
		}
	}

	faceC := face.getPtr()
	*faceC = C.Face{}
	faceC.id = C.FaceId(face.id)
	faceC.state = C.FACESTA_UP
	faceC.numaSocket = C.int(socket.ID())

	sizeofImpl := int(C.sizeof_FaceImpl) + sizeofPriv
	faceC.impl = (*C.FaceImpl)(dpdk.ZmallocAligned("FaceImpl", sizeofImpl, 1, socket))

	return nil

}

// Initialize FaceBase after lower-layer initialization.
// mtu: transport MTU available for NDNLP packets.
// headroom: headroom before NDNLP header, as required by transport.
func (face *FaceBase) FinishInitFaceBase(txQueueCapacity, mtu, headroom int, mempools Mempools) error {
	faceC := face.getPtr()

	r, e := dpdk.NewRing(fmt.Sprintf("FaceTx_%d", face.GetFaceId()),
		txQueueCapacity, face.GetNumaSocket(), false, true)
	if e != nil {
		face.clear()
		return e
	}
	faceC.txQueue = (*C.struct_rte_ring)(r.GetPtr())

	for l3type := 0; l3type < 4; l3type++ {
		latencyStat := running_stat.FromPtr(unsafe.Pointer(&faceC.impl.tx.latency[l3type]))
		latencyStat.Clear(false)
		latencyStat.SetSampleRate(12)
	}

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

func (face *FaceBase) GetFaceId() FaceId {
	return face.id
}

func (face *FaceBase) GetNumaSocket() dpdk.NumaSocket {
	return dpdk.NumaSocketFromID(int(face.getPtr().numaSocket))
}

func (face *FaceBase) IsClosed() bool {
	return face == nil || face.getPtr().id == C.FACEID_INVALID
}

// Prepare to close face.
func (face *FaceBase) BeforeClose() {
	id := face.GetFaceId()
	faceC := face.getPtr()
	faceC.state = C.FACESTA_DOWN
	emitter.EmitSync(evt_FaceClosing, id)
}

func (face *FaceBase) clear() {
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

// Close FaceBase.
// Deallocate FaceImpl.
func (face *FaceBase) CloseFaceBase() {
	id := face.GetFaceId()
	face.clear()
	emitter.EmitSync(evt_FaceClosed, id)
}

func (face *FaceBase) IsDown() bool {
	return face.getPtr().state != C.FACESTA_UP
}

func (face *FaceBase) SetDown(isDown bool) {
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

func (face *FaceBase) TxBurst(pkts []ndn.Packet) {
	if len(pkts) == 0 {
		return
	}
	C.Face_TxBurst(C.FaceId(face.id), (**C.Packet)(unsafe.Pointer(&pkts[0])), C.uint16_t(len(pkts)))
}
