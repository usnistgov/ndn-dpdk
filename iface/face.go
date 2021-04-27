// Package iface implements basics of the face system.
package iface

/*
#include "../csrc/iface/face.h"
*/
import "C"
import (
	"fmt"
	"io"
	"math/rand"
	"unsafe"

	"github.com/pkg/math"
	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/core/logging"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ringbuffer"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"go.uber.org/zap"
	"go4.org/must"
)

var logger = logging.New("iface")

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

// Config contains face configuration.
type Config struct {
	// ReassemblerCapacity is the partial message store capacity in the reassembler.
	//
	// If this value is zero, it defaults to DefaultReassemblerCapacity.
	// Otherwise, it is clamped between MinReassemblerCapacity and MaxReassemblerCapacity.
	ReassemblerCapacity int `json:"reassemblerCapacity,omitempty"`

	// OutputQueueSize is the packet queue capacity before the output thread.
	//
	// The minimum is MinOutputQueueSize.
	// If this value is less than the minimum, it defaults to DefaultOutputQueueSize.
	// Otherwise, it is adjusted up to the next power of 2.
	OutputQueueSize int `json:"outputQueueSize,omitempty"`

	// MTU is the maximum size of outgoing NDNLP packets.
	// This excludes lower layer headers, such as Ethernet/VXLAN headers.
	//
	// Default is the lesser of MaxMTU and what's allowed by network interface and lower layer protocols.
	// If this is less than MinMTU or greater than the maximum, the face will fail to initialize.
	MTU int `json:"mtu,omitempty"`

	maxMTU int
}

// ApplyDefaults applies defaults.
func (c *Config) ApplyDefaults() {
	if c.ReassemblerCapacity == 0 {
		c.ReassemblerCapacity = DefaultReassemblerCapacity
	}
	c.ReassemblerCapacity = math.MinInt(math.MaxInt(MinReassemblerCapacity, c.ReassemblerCapacity), MaxReassemblerCapacity)

	c.OutputQueueSize = ringbuffer.AlignCapacity(c.OutputQueueSize, MinOutputQueueSize, DefaultOutputQueueSize)
}

// WithMaxMTU returns a copy of Config with consideration of device MTU.
func (c Config) WithMaxMTU(max int) Config {
	c.maxMTU = math.MinInt(max, MaxMTU)
	return c
}

func (c *Config) checkMTU() error {
	if c.maxMTU == 0 {
		c.maxMTU = MaxMTU
	}
	if c.MTU == 0 {
		c.MTU = c.maxMTU
	}
	if c.MTU < MinMTU || c.MTU > c.maxMTU {
		return fmt.Errorf("MTU must be between %d and %d", MinMTU, c.maxMTU)
	}
	return nil
}

// NewParams contains parameters to New().
type NewParams struct {
	Config

	// Socket indicates where to allocate memory.
	Socket eal.NumaSocket

	// SizeOfPriv is the size of C.FaceImpl.priv struct.
	SizeofPriv uintptr

	// Init callback is invoked after allocating C.FaceImpl.
	// This is always invoked on the main thread.
	Init func(f Face) (InitResult, error)

	// Start callback is invoked after data structure initialization.
	// It should activate RxGroups associated with the face.
	// It may return a 'subclass' Face interface implementation to make available via Get(id).
	// This is always invoked on the main thread.
	Start func(f Face) (Face, error)

	// Locator callback returns a Locator describing the face.
	Locator func(f Face) Locator

	// Stop callback is invoked to stop the face.
	// It should deactivate RxGroups associated with the face.
	// This is always invoked on the main thread.
	Stop func(f Face) error

	// Close callback is invoked after the face has been removed.
	// This is optional.
	// This is always invoked on the main thread.
	Close func(f Face) error

	// ReadExCounters callback returns extended counters.
	// This is optional.
	ReadExCounters func(f Face) interface{}
}

// InitResult contains results of NewParams.Init callback.
type InitResult struct {
	// TxLinearize indicates whether TX mbufs must be direct mbufs in contiguous memory.
	// See C.PacketTxAlign.linearize field.
	TxLinearize bool

	// L2TxBurst is a C function of C.Face_L2TxBurst type.
	L2TxBurst unsafe.Pointer
}

// New creates a Face.
func New(p NewParams) (face Face, e error) {
	p.Config.ApplyDefaults()
	if e = p.Config.checkMTU(); e != nil {
		return nil, e
	}
	if p.Socket.IsAny() {
		p.Socket = eal.Sockets[rand.Intn(len(eal.Sockets))]
	}

	eal.CallMain(func() {
		face, e = newFace(p)
	})
	return
}

func newFace(p NewParams) (Face, error) {
	f := &face{
		id:                     AllocID(),
		socket:                 p.Socket,
		locatorCallback:        p.Locator,
		stopCallback:           p.Stop,
		closeCallback:          p.Close,
		readExCountersCallback: p.ReadExCounters,
	}
	logEntry := logger.With(
		f.id.ZapField("id"),
		p.Socket.ZapField("socket"),
		zap.Int("mtu", p.MTU),
	)

	c := f.ptr()
	c.id = C.FaceID(f.id)
	c.state = StateUp
	sizeofImpl := C.sizeof_FaceImpl + p.SizeofPriv
	c.impl = (*C.FaceImpl)(eal.ZmallocAligned("FaceImpl", sizeofImpl, 1, p.Socket))

	initResult, e := p.Init(f)
	if e != nil {
		logEntry.Warn("init error", zap.Error(e))
		return f.clear(), e
	}
	logEntry = logEntry.With(zap.Any("locator", f.Locator()))

	c.txAlign = C.PacketTxAlign{
		linearize:           C.bool(initResult.TxLinearize),
		fragmentPayloadSize: C.uint16_t(p.MTU - ndni.LpHeaderHeadroom),
	}
	c.impl.tx.l2Burst = (C.Face_L2TxBurst)(initResult.L2TxBurst)
	(*ndni.Mempools)(unsafe.Pointer(&c.impl.tx.mp)).Assign(p.Socket)

	outputQueue, e := ringbuffer.New(p.OutputQueueSize, p.Socket, ringbuffer.ProducerMulti, ringbuffer.ConsumerSingle)
	if e != nil {
		logEntry.Warn("outputQueue error", zap.Error(e))
		return f.clear(), e
	}
	c.outputQueue = (*C.struct_rte_ring)(outputQueue.Ptr())

	reassID := C.CString(eal.AllocObjectID("iface.Reassembler"))
	defer C.free(unsafe.Pointer(reassID))
	if ok := bool(C.Reassembler_Init(&c.impl.rx.reass, reassID, C.uint32_t(p.ReassemblerCapacity), C.unsigned(p.Socket.ID()))); !ok {
		e := eal.GetErrno()
		logEntry.Warn("Reassembler_Init error", zap.Error(e))
		return f.clear(), e
	}

	C.TxProc_Init(&c.impl.tx, c.txAlign)

	f2, e := p.Start(f)
	if e != nil {
		logEntry.Warn("start error", zap.Error(e))
		return f.clear(), e
	}

	gFaces[f.id] = f2
	emitter.EmitSync(evtFaceNew, f.id)
	ActivateTxFace(f2)
	logEntry.Info("face created")
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
		C.Reassembler_Close(&c.impl.rx.reass)
		eal.Free(c.impl)
	}
	if c.outputQueue != nil {
		must.Close(ringbuffer.FromPtr(unsafe.Pointer(c.outputQueue)))
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
