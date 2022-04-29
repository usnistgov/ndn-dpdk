// Package socketface implements UDP/TCP socket faces using Go net.Conn type.
package socketface

/*
#include "../../csrc/socketface/face.h"
*/
import "C"
import (
	"context"
	"errors"
	"fmt"
	"net"
	"syscall"

	"github.com/usnistgov/ndn-dpdk/core/logging"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
	"github.com/usnistgov/ndn-dpdk/ndn/sockettransport"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"go.uber.org/multierr"
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
	if cfg.MTU > 0 && ndni.PacketMempool.Config().Dataroom < pktmbuf.DefaultHeadroom+cfg.MTU {
		return nil, errors.New("PacketMempool dataroom is too small for requested MTU")
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
	rxi := &rxConnsImpl
	if isUDP && !gCfg.RxEpoll.Disabled {
		rxi = &rxEpollImpl
	}
	txi := &txConnImpl
	if isUDP && !gCfg.TxSyscall.Disabled {
		txi = &txSyscallImpl
	}

	face := &socketFace{
		transport: transport,
	}
	return iface.New(iface.NewParams{
		Config:     cfg.Config.WithMaxMTU(ndni.PacketMempool.Config().Dataroom - pktmbuf.DefaultHeadroom),
		Socket:     gCfg.numaSocket(),
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
			res.TxBurst = txi.txBurst
			return
		},
		Start: func() error {
			if e := rxi.start(face); e != nil {
				face.transport.Close()
				return e
			}

			if txi.start != nil {
				txi.start(face)
			}
			iface.ActivateTxFace(face)

			face.cancelStateChangeHandler = face.transport.OnStateChange(func(st l3.TransportState) {
				face.SetDown(st != l3.TransportUp)
			})

			face.logger.Info("face started", zap.Stringer("rx-impl", rxi), zap.Stringer("tx-impl", txi))
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

func (face *socketFace) rawControl(cb func(ctx context.Context, fd int) error) error {
	raw, e := face.transport.Conn().(syscall.Conn).SyscallConn()
	if e != nil {
		return fmt.Errorf("SyscallConn: %w", e)
	}

	var e1 error
	e0 := raw.Control(func(fd uintptr) {
		e1 = cb(face.transport.Context(), int(fd))
	})
	return multierr.Append(e0, e1)
}
