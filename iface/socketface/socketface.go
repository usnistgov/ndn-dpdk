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

// Config contains socket face configuration.
type Config struct {
	// TxMtu is the maximum size of outgoing NDNLP packets.
	// Zero means unlimited. Otherwise, it is clamped between iface.MinMtu and iface.MaxMtu.
	TxMtu int

	// TxqPkt is the before-TX queue capacity.
	TxqPkts int

	// TxqFrames is the after-TX queue capacity.
	TxqFrames int
}

// New creates a socket face.
func New(loc Locator, cfg Config) (iface.Face, error) {
	if e := loc.Validate(); e != nil {
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

// Wrap wraps a sockettransport.Transport to a socket face.
func Wrap(transport *sockettransport.Transport, cfg Config) (iface.Face, error) {
	face := &socketFace{
		transport: transport,
		rxMempool: ndni.PacketMempool.MakePool(eal.NumaSocket{}),
	}
	return iface.New(iface.NewOptions{
		TxQueueCapacity: cfg.TxqPkts,
		TxMtu:           cfg.TxMtu,
		Init: func(f iface.Face) error {
			face.Face = f
			c := face.ptr()
			c.txBurstOp = (C.FaceImpl_TxBurst)(C.go_SocketFace_TxBurst)
			return nil
		},
		Start: func(f iface.Face) (iface.Face, error) {
			face.transport.OnStateChange(f.SetDown)
			go face.rxLoop()
			if atomic.AddInt32(&nFaces, 1) == 1 {
				iface.EmitRxGroupAdd(rxg)
			}
			return face, nil
		},
		Locator: func(iface.Face) iface.Locator {
			conn := face.transport.Conn()
			laddr, raddr := conn.LocalAddr(), conn.RemoteAddr()

			var loc Locator
			loc.Scheme = raddr.Network()
			loc.Remote = raddr.String()
			if laddr != nil {
				loc.Local = laddr.String()
			}
			return loc
		},
		Stop: func(iface.Face) error {
			if atomic.AddInt32(&nFaces, -1) == 0 {
				iface.EmitRxGroupRemove(rxg)
			}
			return nil
		},
		Close: func(iface.Face) error {
			// close the channel after Get(id) would return nil.
			// Otherwise, go_SocketFace_TxBurst could panic for sending into closed channel.
			close(face.transport.Tx())
			return nil
		},
	})
}

// socketFace is a face using socket as transport.
type socketFace struct {
	iface.Face
	transport *sockettransport.Transport
	rxMempool *pktmbuf.Pool
}

func (face *socketFace) ptr() *C.Face {
	return (*C.Face)(face.Ptr())
}

func (face *socketFace) rxLoop() {
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
	face := iface.Get(iface.ID(faceC.id)).(*socketFace)
	innerTx := face.transport.Tx()
	for i := 0; i < int(nPkts); i++ {
		mbufPtr := (**C.struct_rte_mbuf)(unsafe.Pointer(uintptr(unsafe.Pointer(pkts)) +
			uintptr(i)*unsafe.Sizeof(*pkts)))
		mbuf := pktmbuf.PacketFromPtr(unsafe.Pointer(*mbufPtr))
		wire := mbuf.Bytes()
		mbuf.Close()

		select {
		case innerTx <- wire:
		default: // packet loss
		}
	}
	return nPkts
}
