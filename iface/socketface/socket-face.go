package socketface

/*
#include "socket-face.h"
*/
import "C"
import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"time"
	"unsafe"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
)

const (
	minId = 0xE000
	maxId = 0xEFFF
)

var faceById [maxId - minId + 1]*SocketFace

func getById(id int) *SocketFace {
	return faceById[id-minId]
}

// Retrieve SocketFace by FaceId.
func Get(id iface.FaceId) *SocketFace {
	if id.GetKind() != iface.FaceKind_Socket {
		return nil
	}
	return getById(int(id))
}

func setById(id int, face *SocketFace) {
	faceById[id-minId] = face
}

type Config struct {
	iface.Mempools
	RxMp        dpdk.PktmbufPool // mempool for received frames, dataroom must fit NDNLP frame
	RxqCapacity int              // receive queue length in frames
	TxqCapacity int              // send queue length in frames
}

type SocketFace struct {
	iface.Face
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

func New(conn net.Conn, cfg Config) (face *SocketFace) {
	id := 0
	for {
		id = minId + rand.Intn(maxId-minId+1)
		if getById(id) == nil {
			break
		}
	}

	face = new(SocketFace)
	face.AllocCFace(C.sizeof_SocketFace, dpdk.NUMA_SOCKET_ANY)
	face.logger = log.New(os.Stderr, fmt.Sprintf("face %d ", id), log.LstdFlags)
	face.conn = conn
	face.rxMp = cfg.RxMp
	face.rxQueue = make(chan dpdk.Packet, cfg.RxqCapacity)
	face.rxQuit = make(chan struct{}, 1)
	face.txQueue = make(chan dpdk.Packet, cfg.TxqCapacity)

	C.SocketFace_Init(face.getPtr(), C.FaceId(id),
		(*C.FaceMempools)(cfg.Mempools.GetPtr()))
	setById(id, face)

	if dconn, isDatagram := conn.(net.PacketConn); isDatagram {
		face.impl = newDatagramImpl(face, dconn)
	} else {
		face.impl = newStreamImpl(face, conn)
	}
	go face.impl.RxLoop()
	go face.txLoop()

	return face
}

func (face *SocketFace) close() error {
	face.conn.SetDeadline(time.Now())
	face.rxQuit <- struct{}{}
	close(face.txQueue)
	return face.conn.Close()
}

func (face *SocketFace) getPtr() *C.SocketFace {
	return (*C.SocketFace)(face.GetPtr())
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

func (face *SocketFace) txLoop() {
	for {
		pkt, ok := <-face.txQueue
		if !ok {
			return
		}
		e := face.impl.Send(pkt)
		if e == nil {
			C.FaceImpl_CountSent(&face.getPtr().base, (*C.struct_rte_mbuf)(pkt.GetPtr()))
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

func getByCFace(faceC *C.Face) *SocketFace {
	socketFaceC := (*C.SocketFace)(unsafe.Pointer(faceC))
	face := getById(int(socketFaceC.base.id))
	if face == nil {
		panic("SocketFace not found")
	}
	return face
}

//export go_SocketFace_TxBurst
func go_SocketFace_TxBurst(faceC *C.Face, pkts **C.struct_rte_mbuf, nPkts C.uint16_t) C.uint16_t {
	face := getByCFace(faceC)
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

//export go_SocketFace_Close
func go_SocketFace_Close(faceC *C.Face) C.bool {
	face := getByCFace(faceC)
	e := face.close()
	return e == nil
}
