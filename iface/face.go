package iface

/*
#include "../csrc/iface/face.h"
*/
import "C"
import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/core/runningstat"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/dpdk/ringbuffer"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

// Face is the public API of a face.
// Most functions are implemented by FaceBase type.
type Face interface {
	ptr() *C.Face

	// ID returns ID.
	ID() ID

	// Locator returns a Locator describing face endpoints.
	// Lower layer implementation must provide this method.
	Locator() Locator

	// NumaSocket returns the NUMA socket of this face's data structures.
	NumaSocket() eal.NumaSocket

	// Close destroys the face.
	// Lower layer implementation must provide this method.
	// It should call FaceBase.BeforeClose and FaceBase.CloseFaceBase.
	Close() error

	// IsDown determines whether the face is DOWN or UP.
	IsDown() bool

	// ListRxGroups returns RxGroups that contain this face.
	ListRxGroups() []RxGroup

	// TxBurst transmits a burst of L3 packets.
	TxBurst(pkts []*ndni.Packet)

	// ReadCounters returns basic face counters.
	ReadCounters() Counters

	// ReadExCounters returns extended counters.
	// Lower layer implementation may override this method.
	ReadExCounters() interface{}
}

// FaceBase is a partial implementation of Face interface.
type FaceBase struct {
	id ID
}

func (face *FaceBase) ptr() *C.Face {
	return C.Face_Get(C.FaceID(face.id))
}

// Ptr returns *C.Face pointer.
func (face *FaceBase) Ptr() unsafe.Pointer {
	return unsafe.Pointer(face.ptr())
}

// InitFaceBase should be invoked before lower-layer initialization.
// It allocates FaceImpl on specified NUMA socket.
func (face *FaceBase) InitFaceBase(id ID, sizeofPriv int, socket eal.NumaSocket) error {
	face.id = id

	if socket.IsAny() {
		if lc := eal.CurrentLCore(); lc.Valid() {
			socket = lc.NumaSocket()
		} else {
			socket = eal.Sockets[0]
		}
	}

	faceC := face.ptr()
	*faceC = C.Face{}
	faceC.id = C.FaceID(face.id)
	faceC.state = StateUp
	faceC.numaSocket = C.int(socket.ID())

	sizeofImpl := int(C.sizeof_FaceImpl) + sizeofPriv
	faceC.impl = (*C.FaceImpl)(eal.ZmallocAligned("FaceImpl", sizeofImpl, 1, socket))

	return nil

}

// FinishInitFaceBase should be invoked after lower-layer initialization.
// mtu: transport MTU available for NDNLP packets.
// headroom: headroom before NDNLP header, as required by transport.
func (face *FaceBase) FinishInitFaceBase(txQueueCapacity, mtu, headroom int) error {
	faceC := face.ptr()
	socket := face.NumaSocket()
	indirectMp := pktmbuf.Indirect.MakePool(socket)
	headerMp := ndni.HeaderMempool.MakePool(socket)
	nameMp := ndni.NameMempool.MakePool(socket)

	r, e := ringbuffer.New(txQueueCapacity, socket, ringbuffer.ProducerMulti, ringbuffer.ConsumerSingle)
	if e != nil {
		face.clear()
		return e
	}
	faceC.txQueue = (*C.struct_rte_ring)(r.Ptr())

	for l3type := 0; l3type < 4; l3type++ {
		latencyStat := runningstat.FromPtr(unsafe.Pointer(&faceC.impl.tx.latency[l3type]))
		latencyStat.Clear(false)
		latencyStat.SetSampleRate(12)
	}

	if res := C.TxProc_Init(&faceC.impl.tx, C.uint16_t(mtu), C.uint16_t(headroom),
		(*C.struct_rte_mempool)(indirectMp.Ptr()), (*C.struct_rte_mempool)(headerMp.Ptr())); res != 0 {
		face.clear()
		return eal.Errno(res)
	}

	if res := C.RxProc_Init(&faceC.impl.rx, (*C.struct_rte_mempool)(nameMp.Ptr())); res != 0 {
		face.clear()
		return eal.Errno(res)
	}

	return nil
}

// ID returns ID.
func (face *FaceBase) ID() ID {
	return face.id
}

// NumaSocket returns the NUMA socket of this face's data structures.
func (face *FaceBase) NumaSocket() eal.NumaSocket {
	return eal.NumaSocketFromID(int(face.ptr().numaSocket))
}

// BeforeClose prepares to close face.
func (face *FaceBase) BeforeClose() {
	id := face.ID()
	faceC := face.ptr()
	faceC.state = StateDown
	emitter.EmitSync(evtFaceClosing, id)
}

func (face *FaceBase) clear() {
	id := face.ID()
	faceC := face.ptr()
	faceC.state = StateRemoved
	if faceC.impl != nil {
		eal.Free(faceC.impl)
	}
	if faceC.txQueue != nil {
		ringbuffer.FromPtr(unsafe.Pointer(faceC.txQueue)).Close()
	}
	faceC.id = 0
	gFaces[id] = nil
}

// CloseFaceBase finishes closing face and deallocates FaceImpl.
func (face *FaceBase) CloseFaceBase() {
	id := face.ID()
	face.clear()
	emitter.EmitSync(evtFaceClosed, id)
}

// IsDown determines whether the face is DOWN or UP.
func (face *FaceBase) IsDown() bool {
	return face.ptr().state != StateUp
}

// SetDown changes face state.
func (face *FaceBase) SetDown(isDown bool) {
	if face.IsDown() == isDown {
		return
	}
	id := face.ID()
	faceC := face.ptr()
	if isDown {
		faceC.state = StateDown
		emitter.EmitSync(evtFaceDown, id)
	} else {
		faceC.state = StateUp
		emitter.EmitSync(evtFaceUp, id)
	}
}

// TxBurst transmits a burst of L3 packets.
func (face *FaceBase) TxBurst(pkts []*ndni.Packet) {
	ptr, count := cptr.ParseCptrArray(pkts)
	if count == 0 {
		return
	}
	C.Face_TxBurst(C.FaceID(face.id), (**C.Packet)(ptr), C.uint16_t(count))
}
