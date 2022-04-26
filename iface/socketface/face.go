// Package socketface implements UDP/TCP socket faces using Go net.Conn type.
package socketface

/*
#include "../../csrc/socketface/face.h"
extern uint16_t go_SocketFace_TxBurst(Face* faceC, struct rte_mbuf** pkts, uint16_t nPkts);
STATIC_ASSERT_FUNC_TYPE(Face_TxBurstFunc, go_SocketFace_TxBurst);
*/
import "C"
import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"syscall"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/logging"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
	"github.com/usnistgov/ndn-dpdk/ndn/sockettransport"
	"go.uber.org/zap"
)

var logger = logging.New("socketface")

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
	_, isUDP := transport.Conn().(*net.UDPConn)
	face := &socketFace{
		transport: transport,
	}
	return iface.New(iface.NewParams{
		Config:     cfg.Config,
		SizeofPriv: C.sizeof_SocketFacePriv,
		Init: func(f iface.Face) (res iface.InitResult, e error) {
			face.Face = f
			id, faceC := face.ID(), (*C.Face)(face.Ptr())
			face.logger = logger.With(id.ZapField("id"))

			face.priv = (*C.SocketFacePriv)(C.Face_GetPriv(faceC))
			*face.priv = C.SocketFacePriv{
				fd: -1,
			}

			res.Face = face
			if isUDP {
				res.TxBurst = C.SocketFace_DgramTxBurst
			} else {
				res.TxBurst = C.go_SocketFace_TxBurst
			}
			return
		},
		Start: func() (e error) {
			defer func() {
				if e != nil {
					face.transport.Close()
				}
			}()

			if isUDP {
				e = rxEpollImpl.start(face)
			} else if !gCfg.RxConns.Disabled {
				e = rxConnsImpl.start(face)
			} else {
				e = errors.New("both RxConns and RxEpoll are disabled, cannot start socket face")
			}
			if e != nil {
				return
			}

			if isUDP {
				if e = face.obtainTxFd(); e != nil {
					return
				}
			}
			iface.ActivateTxFace(face)

			face.cancelStateChangeHandler = face.transport.OnStateChange(func(st l3.TransportState) {
				face.SetDown(st != l3.TransportUp)
			})
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
			face.cancelStateChangeHandler()
			face.transport.Close()
			iface.DeactivateTxFace(face)
			return nil
		},
		Close: func() error {
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
	transport                sockettransport.Transport
	logger                   *zap.Logger
	priv                     *C.SocketFacePriv
	cancelStateChangeHandler func()
}

func (face *socketFace) obtainTxFd() error {
	raw, e := face.transport.Conn().(syscall.Conn).SyscallConn()
	if e != nil {
		return fmt.Errorf("SyscallConn: %w", e)
	}

	ready := make(chan struct{})
	go raw.Control(func(fd uintptr) {
		logEntry := face.logger.With(zap.Uintptr("fd", fd))
		logEntry.Debug("file descriptor acquired for socket TX")
		defer logEntry.Debug("file descriptor released for socket TX")
		face.priv.fd = C.int(fd)
		close(ready)
		<-face.transport.Context().Done()
		face.priv.fd = -1
	})
	<-ready
	return nil
}

//export go_SocketFace_TxBurst
func go_SocketFace_TxBurst(faceC *C.Face, pkts **C.struct_rte_mbuf, nPkts C.uint16_t) C.uint16_t {
	face := iface.Get(iface.ID(faceC.id)).(*socketFace)
	vec := pktmbuf.VectorFromPtr(unsafe.Pointer(pkts), int(nPkts))
	defer vec.Close()
	for _, pkt := range vec {
		segs := pkt.SegmentBytes()
		if len(segs) == 1 {
			face.transport.Write(segs[0])
		} else {
			face.transport.Write(bytes.Join(segs, nil))
		}
	}
	return nPkts
}
