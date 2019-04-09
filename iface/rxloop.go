package iface

/*
#include "rxloop.h"

uint16_t go_ChanRxGroup_RxBurst(RxGroup* rxg, struct rte_mbuf** pkts, uint16_t nPkts);
*/
import "C"
import (
	"errors"
	"io"
	"unsafe"

	"ndn-dpdk/core/urcu"
	"ndn-dpdk/dpdk"
)

// Receive channel for a group of faces.
type IRxGroup interface {
	io.Closer

	GetPtr() unsafe.Pointer
	getPtr() *C.RxGroup
	GetRxLoop() *RxLoop
	setRxLoop(rxl *RxLoop)

	GetNumaSocket() dpdk.NumaSocket
	ListFaces() []FaceId
}

// Base type to implement IRxGroup.
type RxGroupBase struct {
	c   unsafe.Pointer
	rxl *RxLoop
}

func (rxg *RxGroupBase) InitRxgBase(c unsafe.Pointer) {
	rxg.c = c
}

func (rxg *RxGroupBase) GetPtr() unsafe.Pointer {
	return rxg.c
}

func (rxg *RxGroupBase) getPtr() *C.RxGroup {
	return (*C.RxGroup)(rxg.c)
}

func (rxg *RxGroupBase) GetRxLoop() *RxLoop {
	return rxg.rxl
}

func (rxg *RxGroupBase) setRxLoop(rxl *RxLoop) {
	rxg.rxl = rxl
}

// An RxGroup using a Go channel as receive queue.
type ChanRxGroup struct {
	RxGroupBase
	faces map[FaceId]IFace
	queue chan dpdk.Packet
}

func newChanRxGroup(queueCapacity int) (rxg *ChanRxGroup) {
	rxg = new(ChanRxGroup)
	C.__theChanRxGroup.rxBurstOp = C.RxGroup_RxBurst(C.go_ChanRxGroup_RxBurst)
	rxg.InitRxgBase(unsafe.Pointer(&C.__theChanRxGroup))
	rxg.faces = make(map[FaceId]IFace)
	rxg.queue = make(chan dpdk.Packet, queueCapacity)
	return rxg
}

func (rxg *ChanRxGroup) Close() error {
	C.free(rxg.GetPtr())
	return nil
}

func (rxg *ChanRxGroup) GetNumaSocket() dpdk.NumaSocket {
	return dpdk.NUMA_SOCKET_ANY
}

func (rxg *ChanRxGroup) ListFaces() (list []FaceId) {
	for faceId := range rxg.faces {
		list = append(list, faceId)
	}
	return list
}

func (rxg *ChanRxGroup) AddFace(face IFace) {
	rxg.faces[face.GetFaceId()] = face
}

func (rxg *ChanRxGroup) RemoveFace(face IFace) {
	delete(rxg.faces, face.GetFaceId())
}

func (rxg *ChanRxGroup) Rx(pkt dpdk.Packet) {
	select {
	case rxg.queue <- pkt:
	default:
		// TODO count drops
		pkt.Close()
	}
}

//export go_ChanRxGroup_RxBurst
func go_ChanRxGroup_RxBurst(rxg *C.RxGroup, pkts **C.struct_rte_mbuf, nPkts C.uint16_t) C.uint16_t {
	select {
	case pkt := <-TheChanRxGroup.queue:
		*pkts = (*C.struct_rte_mbuf)(pkt.GetPtr())
		return 1
	default:
	}
	return 0
}

var TheChanRxGroup = newChanRxGroup(1024)

// RX loop.
type RxLoop struct {
	dpdk.ThreadBase
	c          *C.RxLoop
	numaSocket dpdk.NumaSocket
	rxgs       map[*C.RxGroup]IRxGroup
}

func NewRxLoop(numaSocket dpdk.NumaSocket) (rxl *RxLoop) {
	rxl = new(RxLoop)
	rxl.ResetThreadBase()
	rxl.c = (*C.RxLoop)(dpdk.Zmalloc("RxLoop", C.sizeof_RxLoop, numaSocket))
	dpdk.InitStopFlag(unsafe.Pointer(&rxl.c.stop))
	rxl.numaSocket = numaSocket
	rxl.rxgs = make(map[*C.RxGroup]IRxGroup)
	return rxl
}

func (rxl *RxLoop) GetNumaSocket() dpdk.NumaSocket {
	return rxl.numaSocket
}

func (rxl *RxLoop) SetCallback(cb unsafe.Pointer, cbarg unsafe.Pointer) {
	rxl.c.cb = C.Face_RxCb(cb)
	rxl.c.cbarg = cbarg
}

func (rxl *RxLoop) Launch() error {
	return rxl.LaunchImpl(func() int {
		rs := urcu.NewReadSide()
		defer rs.Close()

		burst := NewRxBurst(64)
		defer burst.Close()
		rxl.c.burst = burst.c

		C.RxLoop_Run(rxl.c)
		return 0
	})
}

func (rxl *RxLoop) Stop() error {
	return rxl.StopImpl(dpdk.NewStopFlag(unsafe.Pointer(&rxl.c.stop)))
}

func (rxl *RxLoop) Close() error {
	if rxl.IsRunning() {
		return dpdk.ErrCloseRunningThread
	}

	for _, rxg := range rxl.rxgs {
		rxg.setRxLoop(nil)
	}

	dpdk.Free(rxl.c)
	return nil
}

func (rxl *RxLoop) ListRxGroups() (list []IRxGroup) {
	for _, rxg := range rxl.rxgs {
		list = append(list, rxg)
	}
	return list
}

func (rxl *RxLoop) ListFaces() (list []FaceId) {
	for _, rxg := range rxl.rxgs {
		list = append(list, rxg.ListFaces()...)
	}
	return list
}

func (rxl *RxLoop) AddRxGroup(rxg IRxGroup) error {
	if rxg.GetRxLoop() != nil {
		return errors.New("RxGroup is active in another RxLoop")
	}
	rxgC := rxg.getPtr()
	if rxgC.rxBurstOp == nil {
		return errors.New("RxGroup.rxBurstOp is missing")
	}

	rs := urcu.NewReadSide()
	defer rs.Close()

	if rxl.numaSocket == dpdk.NUMA_SOCKET_ANY {
		rxl.numaSocket = rxg.GetNumaSocket()
	}
	rxl.rxgs[rxgC] = rxg

	rxg.setRxLoop(rxl)
	C.cds_hlist_add_head_rcu(&rxgC.rxlNode, &rxl.c.head)
	return nil
}

func (rxl *RxLoop) RemoveRxGroup(rxg IRxGroup) error {
	rs := urcu.NewReadSide()
	defer rs.Close()

	rxgC := rxg.getPtr()
	C.cds_hlist_del_rcu(&rxgC.rxlNode)
	urcu.Barrier()

	rxg.setRxLoop(nil)
	delete(rxl.rxgs, rxgC)
	return nil
}
