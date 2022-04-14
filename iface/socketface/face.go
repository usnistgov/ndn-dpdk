// Package socketface implements UDP/TCP socket faces using Go net.Conn type.
package socketface

/*
#include "../../csrc/iface/face.h"
extern uint16_t go_SocketFace_TxBurst(Face* faceC, struct rte_mbuf** pkts, uint16_t nPkts);
STATIC_ASSERT_FUNC_TYPE(Face_TxBurstFunc, go_SocketFace_TxBurst);
*/
import "C"
import (
	"runtime"
	"unsafe"

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

	// sockettransport.Config fields.
	// See ndn-dpdk/ndn/sockettransport package for their semantics and defaults.
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
	dialer.MTU = cfg.MTU
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
	face := &socketFace{
		transport: transport,
		rxMempool: ndni.PacketMempool.Get(eal.NumaSocket{}),
	}
	return iface.New(iface.NewParams{
		Config: cfg.Config,
		Init: func(f iface.Face) (iface.InitResult, error) {
			face.Face = f
			return iface.InitResult{
				Face:    face,
				TxBurst: C.go_SocketFace_TxBurst,
			}, nil
		},
		Start: func() error {
			face.cancelStateChangeHandler = face.transport.OnStateChange(func(st l3.TransportState) {
				face.SetDown(st != l3.TransportUp)
			})

			if e := rxg.addFace(); e != nil {
				return e
			}
			go face.rxLoop()
			iface.ActivateTxFace(face)
			return nil
		},
		Locator: func() iface.Locator {
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
		Stop: func() error {
			if face.cancelStateChangeHandler != nil {
				face.cancelStateChangeHandler()
			}
			rxg.removeFace()
			iface.DeactivateTxFace(face)
			return nil
		},
		Close: func() error {
			face.transport.Close()
			return nil
		},
		ExCounters: func() any {
			return face.transport.Counters()
		},
	})
}

// socketFace is a face using socket as transport.
type socketFace struct {
	iface.Face
	transport sockettransport.Transport
	rxMempool *pktmbuf.Pool

	cancelStateChangeHandler func()
}

func (face *socketFace) rxLoop() {
	for {
		vec, e := face.rxMempool.Alloc(1)
		if e != nil { // ignore alloc error
			runtime.Gosched()
			continue
		}
		pkt := vec[0]
		pkt.SetPort(uint16(face.ID()))
		pkt.SetHeadroom(0)

		for {
			n, e := pkt.ReadFrom(face.transport)
			if e != nil {
				vec.Close()
				return
			}
			if n > 0 {
				break
			}
		}

		pkt.SetTimestamp(eal.TscNow())
		rxg.rx(vec)
	}
}

//export go_SocketFace_TxBurst
func go_SocketFace_TxBurst(faceC *C.Face, pkts **C.struct_rte_mbuf, nPkts C.uint16_t) C.uint16_t {
	face := iface.Get(iface.ID(faceC.id)).(*socketFace)
	vec := pktmbuf.VectorFromPtr(unsafe.Pointer(pkts), int(nPkts))
	defer vec.Close()
	for _, pkt := range vec {
		face.transport.Write(pkt.ZeroCopyBytes())
	}
	return nPkts
}
