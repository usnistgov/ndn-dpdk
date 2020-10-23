// Package socketface implements UDP/TCP socket faces using Go net.Conn type.
package socketface

/*
#include "../../csrc/iface/face.h"
uint16_t go_SocketFace_TxBurst(Face* faceC, struct rte_mbuf** pkts, uint16_t nPkts);
*/
import "C"
import (
	"unsafe"

	"github.com/pkg/math"
	"github.com/usnistgov/ndn-dpdk/core/nnduration"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
	"github.com/usnistgov/ndn-dpdk/ndn/sockettransport"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

// Config contains socket face configuration.
type Config struct {
	iface.Config

	// RxGroupQueueSize is the Go channel buffer size of the RX group channel.
	// Minimum is MinRxGroupQueueSize. Default is DefaultRxGroupQueueSize.
	// This can be changed only if no socket face is present, otherwise this is ignored.
	//
	// The RX group channel is a queue shared among all socket faces, which collects packets received
	// from socket transports, and converts them into DPDK mbufs.
	RxGroupQueueSize int `json:"rxGroupQueueSize,omitempty"`

	// sockettransport.Config fields.
	// See ndn-dpdk/ndn/sockettransport package for their semantics and defaults.
	RxQueueSize          int                     `json:"rxQueueSize,omitempty"`
	TxQueueSize          int                     `json:"txQueueSize,omitempty"`
	RedialBackoffInitial nnduration.Milliseconds `json:"redialBackoffInitial,omitempty"`
	RedialBackoffMaximum nnduration.Milliseconds `json:"redialBackoffMaximum,omitempty"`
}

// New creates a socket face.
func New(loc Locator) (iface.Face, error) {
	if e := loc.Validate(); e != nil {
		return nil, e
	}

	var cfg Config
	if loc.Config != nil {
		cfg = *loc.Config
	}

	var dialer sockettransport.Dialer
	dialer.RxBufferLength = ndni.PacketMempool.Config().Dataroom
	dialer.RxQueueSize = cfg.RxQueueSize
	dialer.TxQueueSize = cfg.TxQueueSize
	dialer.RedialBackoffInitial = cfg.RedialBackoffInitial.Duration()
	dialer.RedialBackoffMaximum = cfg.RedialBackoffMaximum.Duration()
	transport, e := dialer.Dial(loc.Network, loc.Local, loc.Remote)
	if e != nil {
		return nil, e
	}

	return Wrap(transport, cfg)
}

// Wrap wraps a sockettransport.Transport to a socket face.
func Wrap(transport sockettransport.Transport, cfg Config) (iface.Face, error) {
	if cfg.RxGroupQueueSize == 0 {
		cfg.RxGroupQueueSize = DefaultRxGroupQueueSize
	} else {
		cfg.RxGroupQueueSize = math.MaxInt(cfg.RxGroupQueueSize, MinRxGroupQueueSize)
	}

	face := &socketFace{
		transport: transport,
		rxMempool: ndni.PacketMempool.MakePool(eal.NumaSocket{}),
	}
	return iface.New(iface.NewParams{
		Config: cfg.Config,
		Init: func(f iface.Face) error {
			face.Face = f
			c := face.ptr()
			c.txBurstOp = (C.FaceImpl_TxBurst)(C.go_SocketFace_TxBurst)
			return nil
		},
		Start: func(f iface.Face) (iface.Face, error) {
			face.transport.OnStateChange(func(st l3.TransportState) {
				if st == l3.TransportUp {
					f.SetDown(false)
				} else {
					f.SetDown(true)
				}
			})
			if nFaces == 0 {
				rxQueue = make(chan *pktmbuf.Packet, cfg.RxGroupQueueSize)
				iface.EmitRxGroupAdd(rxg)
			}
			go face.rxLoop()
			nFaces++
			return face, nil
		},
		Locator: func(iface.Face) iface.Locator {
			conn := face.transport.Conn()
			laddr, raddr := conn.LocalAddr(), conn.RemoteAddr()

			var loc Locator
			loc.Network = raddr.Network()
			loc.Remote = raddr.String()
			if laddr != nil {
				loc.Local = laddr.String()
			}
			return loc
		},
		Stop: func(iface.Face) error {
			nFaces--
			if nFaces == 0 {
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
		ReadExCounters: func(iface.Face) interface{} {
			return face.transport.Counters()
		},
	})
}

// socketFace is a face using socket as transport.
type socketFace struct {
	iface.Face
	transport sockettransport.Transport
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
