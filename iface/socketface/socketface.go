package socketface

/*
#include "../../csrc/iface/face.h"
uint16_t go_SocketFace_TxBurst(Face* faceC, struct rte_mbuf** pkts, uint16_t nPkts);
*/
import "C"
import (
	"sync/atomic"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn/sockettransport"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

// Config contains SocketFace configuration.
type Config struct {
	TxqPkts   int // before-TX queue capacity
	TxqFrames int // after-TX queue capacity
}

// SocketFace is a face using socket as transport.
type SocketFace struct {
	iface.FaceBase
	transport *sockettransport.Transport
	rxMempool *pktmbuf.Pool
}

// New creates a SocketFace.
func New(loc Locator, cfg Config) (face *SocketFace, e error) {
	if e = loc.Validate(); e != nil {
		return nil, e
	}

	var dialer sockettransport.Dialer
	dialer.RxBufferLength = ndni.PacketMempool.Config().Dataroom
	dialer.TxQueueSize = cfg.TxqFrames
	transport, e := dialer.Dial(loc.Scheme, loc.Local, loc.Remote)
	if e != nil {
		return nil, e
	}

	return Wrap(transport, cfg)
}

// Wrap wraps a sockettransport.Transport to a SocketFace.
func Wrap(transport *sockettransport.Transport, cfg Config) (face *SocketFace, e error) {
	face = new(SocketFace)
	face.rxMempool = ndni.PacketMempool.MakePool(eal.NumaSocket{})
	face.transport = transport

	if e := face.InitFaceBase(iface.AllocID(), 0, eal.NumaSocket{}); e != nil {
		return nil, e
	}

	faceC := face.ptr()
	faceC.txBurstOp = (C.FaceImpl_TxBurst)(C.go_SocketFace_TxBurst)
	if e := face.FinishInitFaceBase(cfg.TxqPkts, 0, 0); e != nil {
		return nil, e
	}

	// face.transport.OnStateChange(func(isDown bool) {
	// 	face.SetDown(isDown)
	// })
	face.transport.OnStateChange(face.SetDown)
	go face.rxLoop()

	if atomic.AddInt32(&nFaces, 1) == 1 {
		iface.EmitRxGroupAdd(rxg)
	}
	iface.Put(face)
	return face, nil
}

func (face *SocketFace) ptr() *C.Face {
	return (*C.Face)(face.Ptr())
}

// Locator returns a locator that describes the socket endpoints.
func (face *SocketFace) Locator() iface.Locator {
	conn := face.transport.Conn()
	laddr, raddr := conn.LocalAddr(), conn.RemoteAddr()

	var loc Locator
	loc.Scheme = raddr.Network()
	loc.Remote = raddr.String()
	if laddr != nil {
		loc.Local = laddr.String()
	}
	return loc
}

// Close destroys the face.
func (face *SocketFace) Close() error {
	face.BeforeClose()
	if atomic.AddInt32(&nFaces, -1) == 0 {
		iface.EmitRxGroupRemove(rxg)
	}
	face.CloseFaceBase()
	close(face.transport.Tx())
	return nil
}

// ListRxGroups returns TheChanRxGroup.
func (face *SocketFace) ListRxGroups() []iface.RxGroup {
	return []iface.RxGroup{rxg}
}

func (face *SocketFace) rxLoop() {
	for {
		wire, ok := <-face.transport.Rx()
		if !ok {
			break
		}

		vec, e := face.rxMempool.Alloc(1)
		if e != nil { // ignore alloc error
			continue
		}

		mbuf := vec[0]
		mbuf.SetPort(uint16(face.ID()))
		mbuf.SetTimestamp(eal.TscNow())
		mbuf.SetHeadroom(0)
		mbuf.Append(wire)

		select {
		case rxQueue <- mbuf:
		default:
			mbuf.Close()
		}
	}
}

//export go_SocketFace_TxBurst
func go_SocketFace_TxBurst(faceC *C.Face, pkts **C.struct_rte_mbuf, nPkts C.uint16_t) C.uint16_t {
	face := iface.Get(iface.ID(faceC.id)).(*SocketFace)
	innerTx := face.transport.Tx()
	for i := 0; i < int(nPkts); i++ {
		mbufPtr := (**C.struct_rte_mbuf)(unsafe.Pointer(uintptr(unsafe.Pointer(pkts)) +
			uintptr(i)*unsafe.Sizeof(*pkts)))
		mbuf := pktmbuf.PacketFromPtr(unsafe.Pointer(*mbufPtr))
		wire := mbuf.ReadAll()
		mbuf.Close()

		select {
		case innerTx <- wire:
		default: // packet loss
		}
	}
	return nPkts
}
