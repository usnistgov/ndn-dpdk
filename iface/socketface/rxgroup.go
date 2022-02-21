package socketface

/*
#include "../../csrc/socketface/rxgroup.h"
*/
import "C"
import (
	"sync"
	"unsafe"

	"github.com/pkg/math"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/dpdk/ringbuffer"
	"github.com/usnistgov/ndn-dpdk/iface"
	"go4.org/must"
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

		if capacity == 0 {
			capacity = DefaultRxGroupCapacity
		} else {
			capacity = math.MaxInt(capacity, MinRxGroupCapacity)
		}
		if rxg.ring, e = ringbuffer.New(capacity, rxg.socket, ringbuffer.ProducerMulti, ringbuffer.ConsumerSingle); e != nil {
			return e
		}

		rxg.c = (*C.SocketRxGroup)(eal.Zmalloc("SocketRxGroup", C.sizeof_SocketRxGroup, eal.NumaSocket{}))
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
	must.Close(rxg.ring)
	rxg.c, rxg.ring = nil, nil
}

func (rxg *rxGroup) rx(vec pktmbuf.Vector) {
	nEnq := rxg.ring.Enqueue(vec)
	for _, m := range vec[nEnq:] {
		must.Close(m)
	}
}
