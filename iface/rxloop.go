package iface

/*
#include "../csrc/iface/rxloop.h"

uint16_t go_ChanRxGroup_RxBurst(RxGroup* rxg, struct rte_mbuf** pkts, uint16_t nPkts);
*/
import "C"
import (
	"errors"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/urcu"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
)

// Receive channel for a group of faces.
type IRxGroup interface {
	Ptr() unsafe.Pointer
	ptr() *C.RxGroup
	GetRxLoop() *RxLoop
	setRxLoop(rxl *RxLoop)

	NumaSocket() eal.NumaSocket
	ListFaces() []ID
}

// Base type to implement IRxGroup.
type RxGroupBase struct {
	c   unsafe.Pointer
	rxl *RxLoop
}

func (rxg *RxGroupBase) InitRxgBase(c unsafe.Pointer) {
	rxg.c = c
}

func (rxg *RxGroupBase) Ptr() unsafe.Pointer {
	return rxg.c
}

func (rxg *RxGroupBase) ptr() *C.RxGroup {
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
	nFaces int32    // accessed via atomic.AddInt32
	faces  sync.Map // map[ID]Face
	queue  chan *pktmbuf.Packet
}

func newChanRxGroup() (rxg *ChanRxGroup) {
	rxg = new(ChanRxGroup)
	C.theChanRxGroup_.rxBurstOp = C.RxGroup_RxBurst(C.go_ChanRxGroup_RxBurst)
	rxg.InitRxgBase(unsafe.Pointer(&C.theChanRxGroup_))
	rxg.SetQueueCapacity(1024)
	return rxg
}

// Change queue capacity (not thread safe).
func (rxg *ChanRxGroup) SetQueueCapacity(queueCapacity int) {
	rxg.queue = make(chan *pktmbuf.Packet, queueCapacity)
}

func (rxg *ChanRxGroup) NumaSocket() eal.NumaSocket {
	return eal.NumaSocket{}
}

func (rxg *ChanRxGroup) ListFaces() (list []ID) {
	rxg.faces.Range(func(faceID, face interface{}) bool {
		list = append(list, faceID.(ID))
		return true
	})
	return list
}

func (rxg *ChanRxGroup) AddFace(face Face) {
	if atomic.AddInt32(&rxg.nFaces, 1) == 1 {
		EmitRxGroupAdd(rxg)
	}
	rxg.faces.Store(face.ID(), face)
}

func (rxg *ChanRxGroup) RemoveFace(face Face) {
	rxg.faces.Delete(face.ID())
	if atomic.AddInt32(&rxg.nFaces, -1) == 0 {
		EmitRxGroupRemove(rxg)
	}
}

func (rxg *ChanRxGroup) Rx(pkt *pktmbuf.Packet) {
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
		*pkts = (*C.struct_rte_mbuf)(pkt.Ptr())
		return 1
	default:
	}
	return 0
}

var TheChanRxGroup = newChanRxGroup()

// RxLoop is a thread to process incoming packets.
type RxLoop struct {
	ealthread.Thread
	c      *C.RxLoop
	socket eal.NumaSocket
	rxgs   map[*C.RxGroup]IRxGroup
}

// NewRxLoop creates an RxLoop.
func NewRxLoop(socket eal.NumaSocket) *RxLoop {
	rxl := &RxLoop{
		c:      (*C.RxLoop)(eal.Zmalloc("RxLoop", C.sizeof_RxLoop, socket)),
		socket: socket,
		rxgs:   make(map[*C.RxGroup]IRxGroup),
	}
	rxl.Thread = ealthread.New(
		rxl.main,
		ealthread.InitStopFlag(unsafe.Pointer(&rxl.c.stop)),
	)
	return rxl
}

// ThreadRole returns "RX" used in lcore allocator.
func (rxl *RxLoop) ThreadRole() string {
	return "RX"
}

// NumaSocket returns NUMA socket of the data structures.
func (rxl *RxLoop) NumaSocket() eal.NumaSocket {
	return rxl.socket
}

// SetCallback assigns a C function and its argument to process received bursts.
func (rxl *RxLoop) SetCallback(cb unsafe.Pointer, cbarg unsafe.Pointer) {
	rxl.c.cb = C.Face_RxCb(cb)
	rxl.c.cbarg = cbarg
}

func (rxl *RxLoop) main() int {
	rs := urcu.NewReadSide()
	defer rs.Close()

	burst := NewRxBurst(64)
	defer burst.Close()
	rxl.c.burst = burst.c

	C.RxLoop_Run(rxl.c)
	return 0
}

// Close stops the thread and deallocates data structures.
func (rxl *RxLoop) Close() error {
	rxl.Stop()

	for _, rxg := range rxl.rxgs {
		rxg.setRxLoop(nil)
	}

	eal.Free(rxl.c)
	return nil
}

func (rxl *RxLoop) ListRxGroups() (list []IRxGroup) {
	for _, rxg := range rxl.rxgs {
		list = append(list, rxg)
	}
	return list
}

func (rxl *RxLoop) ListFaces() (list []ID) {
	for _, rxg := range rxl.rxgs {
		list = append(list, rxg.ListFaces()...)
	}
	return list
}

func (rxl *RxLoop) AddRxGroup(rxg IRxGroup) error {
	if rxg.GetRxLoop() != nil {
		return errors.New("RxGroup is active in another RxLoop")
	}
	rxgC := rxg.ptr()
	if rxgC.rxBurstOp == nil {
		return errors.New("RxGroup.rxBurstOp is missing")
	}

	rs := urcu.NewReadSide()
	defer rs.Close()

	if rxl.socket.IsAny() {
		rxl.socket = rxg.NumaSocket()
	}
	rxl.rxgs[rxgC] = rxg

	rxg.setRxLoop(rxl)
	C.cds_hlist_add_head_rcu(&rxgC.rxlNode, &rxl.c.head)
	return nil
}

func (rxl *RxLoop) RemoveRxGroup(rxg IRxGroup) error {
	rs := urcu.NewReadSide()
	defer rs.Close()

	rxgC := rxg.ptr()
	C.cds_hlist_del_rcu(&rxgC.rxlNode)
	urcu.Barrier()

	rxg.setRxLoop(nil)
	delete(rxl.rxgs, rxgC)
	return nil
}
