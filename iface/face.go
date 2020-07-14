package iface

/*
#include "../csrc/iface/face.h"
*/
import "C"
import (
	"io"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/core/runningstat"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/dpdk/ringbuffer"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

// Face represents a network layer face.
type Face interface {
	eal.WithNumaSocket
	io.Closer

	// Ptr returns *C.Face pointer.
	Ptr() unsafe.Pointer

	// ID returns face ID.
	ID() ID

	// Locator returns a Locator describing face endpoints.
	Locator() Locator

	// ReadCounters returns basic face counters.
	ReadCounters() Counters

	// ReadExCounters returns extended counters.
	ReadExCounters() interface{}

	// SetDown changes face UP/DOWN state.
	SetDown(isDown bool)
}

// NewOptions contains parameters to New().
type NewOptions struct {
	// Socket indicates where to allocate memory.
	Socket eal.NumaSocket

	// SizeOfPriv is the size of C.FaceImpl struct.
	SizeofPriv uintptr

	// TxQueueCapacity is the capacity of the before-TX queue.
	TxQueueCapacity int

	// TxMtu is the maximum size of outgoing NDNLP packets.
	// Zero means unlimited. Otherwise, it is clamped between MinMtu and MaxMtu.
	TxMtu int

	// TxHeadroom is the mbuf headroom to leave before NDNLP header.
	TxHeadroom int

	// Init callback is invoked after allocating C.FaceImpl.
	// It is expected to assign C.Face.txBurstOp.
	Init func(f Face) error

	// Start callback is invoked after data structure initialization.
	// It should activate RxGroups associated with the face.
	// It may return a 'subclass' Face interface implementation to make available via Get(id).
	Start func(f Face) (Face, error)

	// Locator callback returns a Locator describing the face.
	Locator func(f Face) Locator

	// Stop callback is invoked to stop the face.
	// It should deactivate RxGroups associated with the face.
	Stop func(f Face) error

	// Close callback is invoked after the face has been removed.
	// This is optional.
	Close func(f Face) error

	// ReadExCounters callback returns extended counters.
	// This is optional.
	ReadExCounters func(f Face) interface{}
}

// New creates a Face.
func New(p NewOptions) (face Face, e error) {
	eal.CallMain(func() {
		face, e = newFace(p)
	})
	return
}

func newFace(p NewOptions) (Face, error) {
	if p.Socket.IsAny() {
		if lc := eal.CurrentLCore(); lc.Valid() {
			p.Socket = lc.NumaSocket()
		} else {
			p.Socket = eal.Sockets[0]
		}
	}
	f := &face{
		id:                     AllocID(),
		socket:                 p.Socket,
		locatorCallback:        p.Locator,
		stopCallback:           p.Stop,
		closeCallback:          p.Close,
		readExCountersCallback: p.ReadExCounters,
	}

	c := f.ptr()
	c.id = C.FaceID(f.id)
	c.state = StateUp
	sizeofImpl := C.sizeof_FaceImpl + p.SizeofPriv
	c.impl = (*C.FaceImpl)(eal.ZmallocAligned("FaceImpl", sizeofImpl, 1, p.Socket))

	if e := p.Init(f); e != nil {
		return f.clear(), e
	}

	txQueue, e := ringbuffer.New(p.TxQueueCapacity, p.Socket, ringbuffer.ProducerMulti, ringbuffer.ConsumerSingle)
	if e != nil {
		return f.clear(), e
	}
	c.txQueue = (*C.struct_rte_ring)(txQueue.Ptr())

	for l3type := 0; l3type < 4; l3type++ {
		latencyStat := runningstat.FromPtr(unsafe.Pointer(&c.impl.tx.latency[l3type]))
		latencyStat.Clear(false)
		latencyStat.SetSampleRate(12)
	}

	indirectMp := pktmbuf.Indirect.MakePool(p.Socket)
	headerMp := ndni.HeaderMempool.MakePool(p.Socket)

	switch {
	case p.TxMtu == 0:
	case p.TxMtu < MinMtu:
		p.TxMtu = MinMtu
	case p.TxMtu > MaxMtu:
		p.TxMtu = MaxMtu
	}
	C.TxProc_Init(&c.impl.tx, C.uint16_t(p.TxMtu), C.uint16_t(p.TxHeadroom),
		(*C.struct_rte_mempool)(indirectMp.Ptr()), (*C.struct_rte_mempool)(headerMp.Ptr()))

	f2, e := p.Start(f)
	if e != nil {
		return f.clear(), e
	}

	gFaces[f.id] = f2
	emitter.EmitSync(evtFaceNew, f.id)
	ActivateTxFace(f2)
	return f2, nil
}

type face struct {
	id                     ID
	socket                 eal.NumaSocket
	locatorCallback        func(f Face) Locator
	stopCallback           func(f Face) error
	closeCallback          func(f Face) error
	readExCountersCallback func(f Face) interface{}
}

func (f *face) ptr() *C.Face {
	return C.Face_Get(C.FaceID(f.id))
}

func (f *face) Ptr() unsafe.Pointer {
	return unsafe.Pointer(f.ptr())
}

func (f *face) ID() ID {
	return f.id
}

func (f *face) NumaSocket() eal.NumaSocket {
	return f.socket
}

func (f *face) Locator() Locator {
	return f.locatorCallback(f)
}

func (f *face) Close() (e error) {
	eal.CallMain(func() { e = f.close() })
	return e
}

func (f *face) close() error {
	f.ptr().state = StateDown
	emitter.EmitSync(evtFaceClosing, f.id)
	DeactivateTxFace(f)

	if e := f.stopCallback(f); e != nil {
		return e
	}

	f.clear()
	emitter.EmitSync(evtFaceClosed, f.id)

	if f.closeCallback != nil {
		return f.closeCallback(f)
	}
	return nil
}

func (f *face) clear() Face {
	id, c := f.id, f.ptr()
	c.state = StateRemoved
	if c.impl != nil {
		eal.Free(c.impl)
	}
	if c.txQueue != nil {
		ringbuffer.FromPtr(unsafe.Pointer(c.txQueue)).Close()
	}
	c.id = 0
	gFaces[id] = nil
	return nil
}

func (f *face) ReadExCounters() interface{} {
	if f.readExCountersCallback != nil {
		return f.readExCountersCallback(f)
	}
	return nil
}

func (f *face) SetDown(isDown bool) {
	id, c := f.id, f.ptr()
	if IsDown(id) == isDown {
		return
	}
	if isDown {
		c.state = StateDown
		emitter.EmitSync(evtFaceDown, id)
	} else {
		c.state = StateUp
		emitter.EmitSync(evtFaceUp, id)
	}
}

// IsDown returns true if the face does not exist or is down.
func IsDown(id ID) bool {
	return bool(C.Face_IsDown(C.FaceID(id)))
}

// TxBurst transmits a burst of L3 packets.
func TxBurst(id ID, pkts []*ndni.Packet) {
	ptr, count := cptr.ParseCptrArray(pkts)
	C.Face_TxBurst(C.FaceID(id), (**C.Packet)(ptr), C.uint16_t(count))
}
