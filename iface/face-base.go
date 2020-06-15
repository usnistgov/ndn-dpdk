package iface

/*
#include "../csrc/iface/face.h"
*/
import "C"
import (
	"fmt"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/core/runningstat"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/dpdk/ringbuffer"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

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
func (face *FaceBase) InitFaceBase(id FaceId, sizeofPriv int, socket eal.NumaSocket) error {
	face.id = id

	if socket.IsAny() {
		if lc := eal.GetCurrentLCore(); lc.IsValid() {
			socket = lc.GetNumaSocket()
		} else {
			socket = eal.NumaSocketFromID(0) // TODO what if socket 0 is unavailable?
		}
	}

	faceC := face.getPtr()
	*faceC = C.Face{}
	faceC.id = C.FaceId(face.id)
	faceC.state = C.FACESTA_UP
	faceC.numaSocket = C.int(socket.ID())

	sizeofImpl := int(C.sizeof_FaceImpl) + sizeofPriv
	faceC.impl = (*C.FaceImpl)(eal.ZmallocAligned("FaceImpl", sizeofImpl, 1, socket))

	return nil

}

// Initialize FaceBase after lower-layer initialization.
// mtu: transport MTU available for NDNLP packets.
// headroom: headroom before NDNLP header, as required by transport.
func (face *FaceBase) FinishInitFaceBase(txQueueCapacity, mtu, headroom int) error {
	faceC := face.getPtr()
	socket := face.GetNumaSocket()
	indirectMp := pktmbuf.Indirect.MakePool(socket)
	headerMp := ndn.HeaderMempool.MakePool(socket)
	nameMp := ndn.NameMempool.MakePool(socket)

	r, e := ringbuffer.New(fmt.Sprintf("FaceTx_%d", face.GetFaceId()),
		txQueueCapacity, socket, ringbuffer.ProducerMulti, ringbuffer.ConsumerSingle)
	if e != nil {
		face.clear()
		return e
	}
	faceC.txQueue = (*C.struct_rte_ring)(r.GetPtr())

	for l3type := 0; l3type < 4; l3type++ {
		latencyStat := runningstat.FromPtr(unsafe.Pointer(&faceC.impl.tx.latency[l3type]))
		latencyStat.Clear(false)
		latencyStat.SetSampleRate(12)
	}

	if res := C.TxProc_Init(&faceC.impl.tx, C.uint16_t(mtu), C.uint16_t(headroom),
		(*C.struct_rte_mempool)(indirectMp.GetPtr()), (*C.struct_rte_mempool)(headerMp.GetPtr())); res != 0 {
		face.clear()
		return eal.Errno(res)
	}

	if res := C.RxProc_Init(&faceC.impl.rx, (*C.struct_rte_mempool)(nameMp.GetPtr())); res != 0 {
		face.clear()
		return eal.Errno(res)
	}

	return nil
}

func (face *FaceBase) GetFaceId() FaceId {
	return face.id
}

func (face *FaceBase) GetNumaSocket() eal.NumaSocket {
	return eal.NumaSocketFromID(int(face.getPtr().numaSocket))
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
		eal.Free(faceC.impl)
	}
	if faceC.txQueue != nil {
		ringbuffer.FromPtr(unsafe.Pointer(faceC.txQueue)).Close()
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

func (face *FaceBase) TxBurst(pkts []*ndn.Packet) {
	ptr, count := cptr.ParseCptrArray(pkts)
	if count == 0 {
		return
	}
	C.Face_TxBurst(C.FaceId(face.id), (**C.Packet)(ptr), C.uint16_t(count))
}
