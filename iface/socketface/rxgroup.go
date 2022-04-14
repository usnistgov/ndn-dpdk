package socketface

/*
#include "../../csrc/socketface/rxgroup.h"
*/
import "C"
import (
	"sync"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/dpdk/ringbuffer"
	"github.com/usnistgov/ndn-dpdk/iface"
)

const rxGroupCapacity = 4096

type rxGroup struct {
	startOnce sync.Once
	cmd       chan func()
	nFaces    int

	socket eal.NumaSocket
	ring   *ringbuffer.Ring
	c      *C.SocketRxGroup
}

var _ iface.RxGroup = (*rxGroup)(nil)
var rxg = &rxGroup{}

func (rxg *rxGroup) NumaSocket() eal.NumaSocket {
	return rxg.socket
}

func (rxg *rxGroup) RxGroup() (ptr unsafe.Pointer, desc string) {
	return unsafe.Pointer(rxg.c), "SocketRxGroup"
}

func (rxg *rxGroup) loop() {
	for f := range rxg.cmd {
		f()
	}
}

func (rxg *rxGroup) activate() (e error) {
	rxg.socket = eal.RandomSocket()
	if rxg.ring, e = ringbuffer.New(rxGroupCapacity, rxg.socket, ringbuffer.ProducerMulti, ringbuffer.ConsumerSingle); e != nil {
		return e
	}

	rxg.c = eal.Zmalloc[C.SocketRxGroup]("SocketRxGroup", C.sizeof_SocketRxGroup, rxg.socket)
	rxg.c.base.rxBurst = C.RxGroup_RxBurstFunc(C.SocketRxGroup_RxBurst)
	rxg.c.ring = (*C.struct_rte_ring)(rxg.ring.Ptr())

	iface.ActivateRxGroup(rxg)
	return nil
}

func (rxg *rxGroup) deactivate() {
	iface.DeactivateRxGroup(rxg)
	eal.Free(rxg.c)
	rxg.ring.Close()
	rxg.c, rxg.ring = nil, nil
}

func (rxg *rxGroup) addFace() (e error) {
	rxg.startOnce.Do(func() {
		rxg.cmd = make(chan func())
		go rxg.loop()
	})

	done := make(chan struct{})
	rxg.cmd <- func() {
		defer close(done)
		rxg.nFaces++
		if rxg.nFaces == 1 {
			e = rxg.activate()
		}
	}
	<-done
	return
}

func (rxg *rxGroup) removeFace() {
	rxg.cmd <- func() {
		rxg.nFaces--
		if rxg.nFaces == 0 {
			rxg.deactivate()
		}
	}
}

func (rxg *rxGroup) rx(vec pktmbuf.Vector) {
	rxg.cmd <- func() {
		var nEnq int
		if rxg.ring != nil {
			nEnq = ringbuffer.Enqueue(rxg.ring, vec)
		}
		vec[nEnq:].Close()
	}
}
