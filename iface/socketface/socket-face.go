package socketface

/*
#include "socket-face.h"
*/
import "C"
import (
	"math/rand"
	"net"
	"unsafe"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

var faceById = make(map[int]*SocketFace)

const (
	minId = 0xE000
	maxId = 0xEFFF
)

type Config struct {
	RxCapacity int              // receive queue length in packets
	TxCapacity int              // send queue length in packets
	RxMp       dpdk.PktmbufPool // mempool for received packets
}

type SocketFace struct {
	iface.Face
	conn net.Conn

	txQueue        chan ndn.Packet // TX queue
	txQuit         chan struct{}   // stop TxLoop
	nTxCongestions int             // number of incomplete TX bursts due to queue congestion
}

type iImpl interface {
	TxLoop(face *SocketFace)
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
	face.conn = conn
	face.txQueue = make(chan ndn.Packet, cfg.TxCapacity)

	C.SocketFace_Init(face.getPtr(), C.uint16_t(id))
	faceById[id] = face

	var impl iImpl
	if _, isDatagram := conn.(net.PacketConn); isDatagram {
		panic("datagram not implemented")
	} else {
		impl = streamImpl{}
	}
	go impl.TxLoop(face)

	return face
}

func (face SocketFace) getPtr() *C.SocketFace {
	return (*C.SocketFace)(face.GetPtr())
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
	_ = getByCFace(faceC)
	panic("RxBurst not implemented")
	return 0
}

//export go_SocketFace_TxBurst
func go_SocketFace_TxBurst(faceC *C.Face, pkts **C.struct_rte_mbuf, nPkts C.uint16_t) {
	face := getByCFace(faceC)

L:
	for i := 0; i < int(nPkts); i++ {
		pktsEle := (**C.struct_rte_mbuf)(unsafe.Pointer(uintptr(unsafe.Pointer(pkts)) +
			uintptr(i)*unsafe.Sizeof(*pkts)))
		pkt := ndn.PacketFromPtr(unsafe.Pointer(*pktsEle))
		select {
		case face.txQueue <- pkt:
		default:
			face.nTxCongestions++
			break L
		}
	}
}

//export go_SocketFace_Close
func go_SocketFace_Close(faceC *C.Face) C.bool {
	face := getByCFace(faceC)
	e := face.conn.Close()
	face.txQuit <- struct{}{}
	return e == nil
}
