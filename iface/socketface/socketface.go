package socketface

/*
#include "../../csrc/iface/face.h"
uint16_t go_SocketFace_TxBurst(Face* faceC, struct rte_mbuf** pkts, uint16_t nPkts);
*/
import "C"
import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
	nsf "github.com/usnistgov/ndn-dpdk/ndn/socketface"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
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
	inner     *nsf.SocketFace
	rxMempool *pktmbuf.Pool
}

// New creates a SocketFace.
func New(loc Locator, cfg Config) (face *SocketFace, e error) {
	if e = loc.Validate(); e != nil {
		return nil, e
	}

	var dialer nsf.Dialer
	dialer.RxBufferLength = ndni.PacketMempool.GetConfig().Dataroom
	dialer.TxQueueSize = cfg.TxqFrames
	inner, e := dialer.Dial(loc.Scheme, loc.Local, loc.Remote)
	if e != nil {
		return nil, e
	}

	return Wrap(inner, cfg)
}

// Wrap wraps a nsf.SocketFace to a SocketFace.
func Wrap(inner *nsf.SocketFace, cfg Config) (face *SocketFace, e error) {
	face = new(SocketFace)
	face.rxMempool = ndni.PacketMempool.MakePool(eal.NumaSocket{})
	face.inner = inner

	if e := face.InitFaceBase(iface.AllocId(iface.FaceKind_Socket), 0, eal.NumaSocket{}); e != nil {
		return nil, e
	}

	faceC := face.getPtr()
	faceC.txBurstOp = (C.FaceImpl_TxBurst)(C.go_SocketFace_TxBurst)
	if e := face.FinishInitFaceBase(cfg.TxqPkts, 0, 0); e != nil {
		return nil, e
	}

	face.inner.OnStateChange(func(isDown bool) {
		face.SetDown(isDown)
	})

	go face.rxLoop()
	iface.TheChanRxGroup.AddFace(face)
	iface.Put(face)
	return face, nil
}

func (face *SocketFace) getPtr() *C.Face {
	return (*C.Face)(face.GetPtr())
}

// GetLocator returns a locator that describes the socket endpoints.
func (face *SocketFace) GetLocator() iface.Locator {
	conn := face.inner.GetConn()
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
	e := face.inner.Close()
	iface.TheChanRxGroup.RemoveFace(face)
	face.CloseFaceBase()
	return e
}

// ListRxGroups returns TheChanRxGroup.
func (face *SocketFace) ListRxGroups() []iface.IRxGroup {
	return []iface.IRxGroup{iface.TheChanRxGroup}
}

func (face *SocketFace) rxLoop() {
	for {
		packet, ok := <-face.inner.GetRx()
		if !ok {
			break
		}

		wire, e := tlv.Encode(packet)
		if e != nil { // ignore encoding error
			continue
		}

		vec, e := face.rxMempool.Alloc(1)
		if e != nil { // ignore alloc error
			continue
		}

		mbuf := vec[0]
		mbuf.SetPort(uint16(face.GetFaceId()))
		mbuf.SetTimestamp(eal.TscNow())
		mbuf.SetHeadroom(0)
		mbuf.Append(wire)
		iface.TheChanRxGroup.Rx(mbuf)
	}
}

//export go_SocketFace_TxBurst
func go_SocketFace_TxBurst(faceC *C.Face, pkts **C.struct_rte_mbuf, nPkts C.uint16_t) C.uint16_t {
	face := iface.Get(iface.FaceId(faceC.id)).(*SocketFace)
	for i := 0; i < int(nPkts); i++ {
		mbufPtr := (**C.struct_rte_mbuf)(unsafe.Pointer(uintptr(unsafe.Pointer(pkts)) +
			uintptr(i)*unsafe.Sizeof(*pkts)))
		mbuf := pktmbuf.PacketFromPtr(unsafe.Pointer(*mbufPtr))
		wire := mbuf.ReadAll()
		mbuf.Close()

		var packet ndn.Packet
		e := tlv.Decode(wire, &packet)
		if e != nil {
			continue
		}
		select {
		case face.inner.GetTx() <- &packet:
		default: // packet loss
		}
	}
	return nPkts
}
