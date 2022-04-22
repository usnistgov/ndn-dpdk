package socketface

/*
#include "../../csrc/socketface/rxepoll.h"
*/
import "C"
import (
	"fmt"
	"syscall"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
)

type rxEpoll struct {
	socket eal.NumaSocket
	epfd   int
	c      *C.SocketRxEpoll
}

func (rxe *rxEpoll) NumaSocket() eal.NumaSocket {
	return rxe.socket
}

func (rxe *rxEpoll) RxGroup() (ptr unsafe.Pointer, desc string) {
	return unsafe.Pointer(rxe.c), "SocketRxEpoll"
}

func (rxe *rxEpoll) close() {
	iface.DeactivateRxGroup(rxe)
	unix.Close(rxe.epfd)
	eal.Free(rxe.c)
	rxe.c, rxe.epfd = nil, -1
	logger.Debug("RxEpoll closed")
}

func (rxe *rxEpoll) run(face *socketFace) error {
	ctx := face.transport.Context()
	raw, e := face.transport.Conn().(syscall.Conn).SyscallConn()
	if e != nil {
		return fmt.Errorf("SyscallConn: %w", e)
	}

	var e1 error
	e0 := raw.Control(func(fd uintptr) {
		logEntry := face.logger.With(zap.Int("epfd", rxe.epfd), zap.Uintptr("fd", fd))
		logEntry.Debug("file descriptor acquired for RxEpoll")
		defer logEntry.Debug("file descriptor released for RxEpoll")

		var event unix.EpollEvent
		C.SocketRxEpoll_PrepareEvent((*C.struct_epoll_event)(unsafe.Pointer(&event)), C.FaceID(face.ID()), C.int(fd))
		if e := unix.EpollCtl(rxe.epfd, unix.EPOLL_CTL_ADD, int(fd), &event); e != nil {
			e1 = fmt.Errorf("unix.EpollCtl(EPOLL_CTL_ADD): %w", e)
			return
		}

		<-ctx.Done()

		if e := unix.EpollCtl(rxe.epfd, unix.EPOLL_CTL_DEL, int(fd), nil); e != nil {
			e1 = fmt.Errorf("unix.EpollCtl(EPOLL_CTL_DEL): %w", e)
		}
	})
	return multierr.Append(e0, e1)
}

func newRxEpoll(socket eal.NumaSocket) (rxe *rxEpoll, e error) {
	epfd, e := unix.EpollCreate1(0)
	if e != nil {
		return nil, fmt.Errorf("unix.EpollCreate1: %w", e)
	}

	rxe = &rxEpoll{
		epfd:   epfd,
		socket: eal.RewriteAnyNumaSocketFirst.Rewrite(socket),
	}
	rxe.c = eal.Zmalloc[C.SocketRxEpoll]("SocketRxEpoll", C.sizeof_SocketRxEpoll, rxe.socket)
	rxe.c.base.rxBurst = C.RxGroup_RxBurstFunc(C.SocketRxEpoll_RxBurst)
	rxe.c.directMp = (*C.struct_rte_mempool)(ndni.PacketMempool.Get(rxe.socket).Ptr())
	rxe.c.epfd = C.int(epfd)

	logger.Debug("RxEpoll created",
		zap.Int("epfd", rxe.epfd),
		rxe.socket.ZapField("socket"),
	)
	iface.ActivateRxGroup(rxe)
	return rxe, nil
}

var rxEpollImpl = rxImpl{
	nilValue: (*rxEpoll)(nil),
	create: func() (rxGroup, error) {
		return newRxEpoll(gCfg.RxEpoll.Socket)
	},
}
