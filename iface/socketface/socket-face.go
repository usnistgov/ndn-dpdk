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
	RxMp        dpdk.PktmbufPool // mempool for received packets, dataroom must be at least MTU
	RxqCapacity int              // receive queue length in packets
	TxqCapacity int              // send queue length in packets
}

type SocketFace struct {
	iface.Face
	logger *log.Logger
	conn   net.Conn

	rxMp    dpdk.PktmbufPool
	rxQueue chan ndn.Packet // RX queue

	txQueue        chan ndn.Packet // TX queue
	txQuit         chan struct{}   // stop TxLoop
	nTxCongestions int             // number of incomplete TX bursts due to queue congestion
}

type iImpl interface {
	RxLoop(face *SocketFace)
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
	face.logger = log.New(os.Stderr, fmt.Sprintf("face %d ", id), log.LstdFlags)
	face.conn = conn
	face.rxMp = cfg.RxMp
	face.rxQueue = make(chan ndn.Packet, cfg.RxqCapacity)
	face.txQueue = make(chan ndn.Packet, cfg.TxqCapacity)
	face.txQuit = make(chan struct{}, 1)

	C.SocketFace_Init(face.getPtr(), C.uint16_t(id))
	faceById[id] = face

	var impl iImpl
	if _, isDatagram := conn.(net.PacketConn); isDatagram {
		impl = datagramImpl{}
	} else {
		impl = streamImpl{}
	}
	go impl.RxLoop(face)
	go impl.TxLoop(face)

	return face
}

func (face *SocketFace) Close() error {
	return face.conn.Close()
}

func (face *SocketFace) getPtr() *C.SocketFace {
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
	return e == nil
}

//export go_SocketFace_ReadCounters
func go_SocketFace_ReadCounters(faceC *C.Face, cntC *C.FaceCounters) {
	// face := getByCFace(faceC)
	cnt := (*iface.Counters)(unsafe.Pointer(cntC))
	cnt.RxL2.NFrames = 1
	cnt.TxL2.NFrames = 1
	// TODO implement counters
}
