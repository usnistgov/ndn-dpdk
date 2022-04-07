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

// Limits of RxGroupCapacity.
const (
	MinRxGroupCapacity     = 256
	DefaultRxGroupCapacity = 4096
)

type rxGroup struct {
	mutex  sync.Mutex
	nFaces int

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

func (rxg *rxGroup) addFace(capacity int) (e error) {
	rxg.mutex.Lock()
	defer rxg.mutex.Unlock()

	if rxg.nFaces == 0 {
		rxg.socket = eal.RandomSocket()
		capacity = ringbuffer.AlignCapacity(capacity, MinRxGroupCapacity, DefaultRxGroupCapacity)
		if rxg.ring, e = ringbuffer.New(capacity, rxg.socket, ringbuffer.ProducerMulti, ringbuffer.ConsumerSingle); e != nil {
			return e
		}

		rxg.c = eal.Zmalloc[C.SocketRxGroup]("SocketRxGroup", C.sizeof_SocketRxGroup, rxg.socket)
		rxg.c.base.rxBurst = C.RxGroup_RxBurstFunc(C.SocketRxGroup_RxBurst)
		rxg.c.ring = (*C.struct_rte_ring)(rxg.ring.Ptr())

		iface.ActivateRxGroup(rxg)
	}

	rxg.nFaces++
	return nil
}

func (rxg *rxGroup) removeFace() {
	rxg.mutex.Lock()
	defer rxg.mutex.Unlock()

	rxg.nFaces--
	if rxg.nFaces > 0 {
		return
	}

	iface.DeactivateRxGroup(rxg)
	eal.Free(rxg.c)
	rxg.ring.Close()
	rxg.c, rxg.ring = nil, nil
}

func (rxg *rxGroup) rx(vec pktmbuf.Vector) {
	nEnq := ringbuffer.Enqueue(rxg.ring, vec)
	vec[nEnq:].Close()
}
