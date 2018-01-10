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

var faceById = make(map[int]*SocketFace)

const (
	minId = 0xE000
	maxId = 0xEFFF
)

type Config struct {
	RxMp        dpdk.PktmbufPool // mempool for received frames, dataroom must fit NDNLP frame
	RxqCapacity int              // receive queue length in frames

	TxIndirectMp dpdk.PktmbufPool // mempool for indirect mbufs
	TxHeaderMp   dpdk.PktmbufPool // mempool for NDNLP header
	TxqCapacity  int              // send queue length in frames
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
	// Receive one NDNLP packet on the socket.
	Recv() ([]byte, error)

	// Transmit one packet on the socket.
	Send(pkt dpdk.Packet) error
}

func New(conn net.Conn, cfg Config) (face *SocketFace) {
	id := 0
	for {
		id = minId + rand.Intn(maxId-minId+1)
		if _, hasOldFace := faceById[id]; !hasOldFace {
			break
		}
	}

	face = new(SocketFace)
	face.Face = iface.FaceFromPtr(C.calloc(1, C.sizeof_SocketFace))
	face.logger = log.New(os.Stderr, fmt.Sprintf("face %d ", id), log.LstdFlags)
	face.conn = conn
	face.rxMp = cfg.RxMp
	face.rxQueue = make(chan dpdk.Packet, cfg.RxqCapacity)
	face.rxQuit = make(chan struct{})
	face.txQueue = make(chan dpdk.Packet, cfg.TxqCapacity)

	C.SocketFace_Init(face.getPtr(), C.uint16_t(id),
		(*C.struct_rte_mempool)(cfg.TxIndirectMp.GetPtr()),
		(*C.struct_rte_mempool)(cfg.TxHeaderMp.GetPtr()))
	faceById[id] = face

	if dconn, isDatagram := conn.(net.PacketConn); isDatagram {
		face.impl = newDatagramImpl(face, dconn)
	} else {
		face.impl = newStreamImpl(face, conn)
	}
	go face.RxLoop()
	go face.TxLoop()

	return face
}

func (face *SocketFace) Close() error {
	face.conn.SetDeadline(time.Now())
	face.rxQuit <- struct{}{}
	close(face.txQueue)
	return face.conn.Close()
}

func (face *SocketFace) getPtr() *C.SocketFace {
	return (*C.SocketFace)(face.GetPtr())
}

func (face *SocketFace) RxLoop() {
	for {
		buf, e := face.impl.Recv()
		if face.handleError("RX", e) {
			return
		}

		mbuf, e := face.rxMp.Alloc()
		if e != nil {
			face.logger.Printf("RX alloc error: %v", e)
			continue
		}

		pkt := mbuf.AsPacket()
		seg0 := pkt.GetFirstSegment()
		seg0.SetHeadroom(0)
		seg0.AppendOctets(buf[:])

		select {
		case <-face.rxQuit:
			pkt.Close()
			return
		case face.rxQueue <- pkt:
		default:
			pkt.Close()
			face.rxCongestions++
			face.logger.Printf("RX queue is full, %d", face.rxCongestions)
		}
	}
}

func (face *SocketFace) TxLoop() {
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
		if face.handleError("TX", e) {
			return
		}
	}
}

// Handle socket error, if any. Return whether RxLoop or TxLoop should terminate.
func (face *SocketFace) handleError(dir string, e error) bool {
	if e == nil {
		return false
	}
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
	face, ok := faceById[int(socketFaceC.base.id)]
	if !ok {
		panic("SocketFace not found")
	}
	return face
}

//export go_SocketFace_RxBurst
func go_SocketFace_RxBurst(faceC *C.Face, pkts **C.struct_rte_mbuf, nPkts C.uint16_t) C.uint16_t {
	face := getByCFace(faceC)
	nReceived := C.uint16_t(0)
	for i := C.uint16_t(0); i < nPkts; i++ {
		pktsEle := (**C.struct_rte_mbuf)(unsafe.Pointer(uintptr(unsafe.Pointer(pkts)) +
			uintptr(i)*unsafe.Sizeof(*pkts)))
		select {
		case pkt := <-face.rxQueue:
			*pktsEle = (*C.struct_rte_mbuf)(pkt.GetPtr())
			nReceived++
		default:
			return nReceived
		}
	}
	return nReceived
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
	e := face.Close()
	return e == nil
}
