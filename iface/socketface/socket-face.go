package socketface

/*
#include "../face.h"
uint16_t go_SocketFace_TxBurst(Face* faceC, struct rte_mbuf** pkts, uint16_t nPkts);
*/
import "C"
import (
	"fmt"
	"log"
	"net"
	"os"
	"time"
	"unsafe"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
)

type Config struct {
	iface.Mempools
	RxMp        dpdk.PktmbufPool // mempool for received frames, dataroom must fit NDNLP frame
	RxqCapacity int              // receive queue length in frames
	TxqCapacity int              // send queue length in frames
}

type SocketFace struct {
	iface.BaseFace
	logger *log.Logger
	conn   net.Conn
	impl   impl
	failed bool

	rxMp          dpdk.PktmbufPool
	rxQueue       chan dpdk.Packet
	rxQuit        chan struct{}
	rxCongestions int // L2 frames dropped due to rxQueue full

	txQueue chan dpdk.Packet
}

type impl interface {
	// Receive packets on the socket and post them to face.rxQueue.
	// Loop until a fatal error occurs or face.rxQuit receives a message.
	// Increment face.rxCongestions when a packet arrives but face.rxQueue is full.
	RxLoop()

	// Transmit one packet on the socket.
	Send(pkt dpdk.Packet) error
}

func New(conn net.Conn, cfg Config) *SocketFace {
	var face SocketFace
	face.InitBaseFace(iface.AllocId(iface.FaceKind_Socket), 0, dpdk.NUMA_SOCKET_ANY)

	face.logger = log.New(os.Stderr, fmt.Sprintf("face %d ", face.GetFaceId()), log.LstdFlags)
	face.conn = conn
	face.rxMp = cfg.RxMp
	face.rxQueue = make(chan dpdk.Packet, cfg.RxqCapacity)
	face.rxQuit = make(chan struct{}, 1)
	face.txQueue = make(chan dpdk.Packet, cfg.TxqCapacity)

	if dconn, isDatagram := conn.(net.PacketConn); isDatagram {
		face.impl = newDatagramImpl(&face, dconn)
	} else {
		face.impl = newStreamImpl(&face, conn)
	}
	go face.impl.RxLoop()
	go face.txLoop()

	faceC := face.getPtr()
	faceC.txBurstOp = (C.FaceImpl_TxBurst)(C.go_SocketFace_TxBurst)
	C.FaceImpl_Init(faceC, 0, 0, (*C.FaceMempools)(cfg.Mempools.GetPtr()))
	iface.Put(&face)
	return &face
}

func (face *SocketFace) getPtr() *C.Face {
	return (*C.Face)(face.GetPtr())
}

func (face *SocketFace) Close() error {
	face.conn.SetDeadline(time.Now())
	face.rxQuit <- struct{}{}
	close(face.txQueue)
	return face.conn.Close()
}

func (face *SocketFace) rxBurst(burst iface.RxBurst) (nRx int) {
	capacity := burst.GetCapacity()
	for ; nRx < capacity; nRx++ {
		select {
		case pkt := <-face.rxQueue:
			burst.SetFrame(nRx, pkt)
		default:
			return nRx
		}
	}
	return nRx
}

// Report congestion when RxLoop is unable to send into rxQueue.
func (face *SocketFace) rxReportCongestion() {
	face.rxCongestions++
	if face.rxCongestions%1024 == 0 {
		face.logger.Printf("RX queue is full, %d", face.rxCongestions)
	}
}

func (face *SocketFace) txLoop() {
	for {
		pkt, ok := <-face.txQueue
		if !ok {
			return
		}
		e := face.impl.Send(pkt)
		if e == nil {
			C.FaceImpl_CountSent(face.getPtr(), (*C.struct_rte_mbuf)(pkt.GetPtr()))
		}
		pkt.Close()
		if e != nil && face.handleError("TX", e) {
			return
		}
	}
}

// Handle socket error.
// Return whether RxLoop or TxLoop should terminate.
func (face *SocketFace) handleError(dir string, e error) bool {
	if netErr, ok := e.(net.Error); ok && netErr.Temporary() {
		face.logger.Printf("%s socket error: %v", dir, e)
		return false
	}
	face.logger.Printf("%s socket failed: %v", dir, e)
	face.conn.Close()
	face.failed = true
	return true
}

//export go_SocketFace_TxBurst
func go_SocketFace_TxBurst(faceC *C.Face, pkts **C.struct_rte_mbuf, nPkts C.uint16_t) C.uint16_t {
	face := iface.Get(iface.FaceId(faceC.id)).(*SocketFace)
	nQueued := C.uint16_t(0)
	for i := C.uint16_t(0); i < nPkts; i++ {
		pktsEle := (**C.struct_rte_mbuf)(unsafe.Pointer(uintptr(unsafe.Pointer(pkts)) +
			uintptr(i)*unsafe.Sizeof(*pkts)))
		pkt := dpdk.MbufFromPtr(unsafe.Pointer(*pktsEle)).AsPacket()
		select {
		case face.txQueue <- pkt:
			nQueued++
		default:
			return nQueued
		}
	}
	return nQueued
}
