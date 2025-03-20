package socketface

/*
#include "../../csrc/socketface/rxepoll.h"
*/
import "C"
import (
	"context"
	"fmt"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
)

type rxEpoll struct {
	rxFaceList
	socket eal.NumaSocket
	epfd   int
	c      *C.SocketRxEpoll
}

var _ iface.RxGroup = &rxEpoll{}

func (rxe *rxEpoll) NumaSocket() eal.NumaSocket {
	return rxe.socket
}

func (rxe *rxEpoll) RxGroup() (ptr unsafe.Pointer, desc string) {
	return unsafe.Pointer(rxe.c), "SocketRxEpoll"
}

func (rxe *rxEpoll) close() {
	iface.DeactivateRxGroup(rxe)
	pktmbuf.VectorFromPtr(unsafe.Pointer(&rxe.c.mbufs), len(rxe.c.msgs))[rxe.c.msgIndex:].Close()
	unix.Close(rxe.epfd)
	eal.Free(rxe.c)
	rxe.c, rxe.epfd = nil, -1
	logger.Info("RxEpoll closed")
}

func (rxe *rxEpoll) run(face *socketFace) error {
	defer rxe.faceListPut(face)()
	return face.rawControl(func(ctx context.Context, fd int) error {
		logEntry := face.logger.With(zap.Int("epfd", rxe.epfd), zap.Int("fd", fd))
		logEntry.Debug("file descriptor acquired for RxEpoll")
		defer logEntry.Debug("file descriptor released for RxEpoll")

		var event unix.EpollEvent
		C.SocketRxEpoll_PrepareEvent((*C.struct_epoll_event)(unsafe.Pointer(&event)), C.FaceID(face.ID()), C.int(fd))
		if e := unix.EpollCtl(rxe.epfd, unix.EPOLL_CTL_ADD, int(fd), &event); e != nil {
			return fmt.Errorf("unix.EpollCtl(ADD,%d): %w", fd, e)
		}

		<-ctx.Done()

		if e := unix.EpollCtl(rxe.epfd, unix.EPOLL_CTL_DEL, int(fd), nil); e != nil {
			return fmt.Errorf("unix.EpollCtl(DEL,%d): %w", fd, e)
		}
		return nil
	})
}

func newRxEpoll(socket eal.NumaSocket) (rxe *rxEpoll, e error) {
	epfd, e := unix.EpollCreate1(0)
	if e != nil {
		return nil, fmt.Errorf("unix.EpollCreate1: %w", e)
	}

	rxe = &rxEpoll{
		epfd:   epfd,
		socket: socket,
	}
	rxe.c = eal.Zmalloc[C.SocketRxEpoll]("SocketRxEpoll", C.sizeof_SocketRxEpoll, rxe.socket)
	rxe.c.base.rxBurst = C.RxGroup_RxBurstFunc(C.SocketRxEpoll_RxBurst)
	rxe.c.directMp = (*C.struct_rte_mempool)(ndni.PacketMempool.Get(rxe.socket).Ptr())
	rxe.c.epfd = C.int(epfd)
	rxe.c.msgIndex = C.uint16_t(len(rxe.c.msgs))

	logger.Info("RxEpoll created",
		zap.Int("epfd", rxe.epfd),
		rxe.socket.ZapField("socket"),
	)
	iface.ActivateRxGroup(rxe)
	return rxe, nil
}

var rxEpollImpl = rxImpl{
	describe: "RxEpoll",
	nilValue: (*rxEpoll)(nil),
	create: func() (rxGroup, error) {
		return newRxEpoll(gCfg.numaSocket())
	},
}
