package socketface

/*
#include "../../csrc/socketface/rxconns.h"
*/
import "C"
import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/dpdk/ringbuffer"
	"github.com/usnistgov/ndn-dpdk/iface"
	"go.uber.org/atomic"
)

type rxConns struct {
	c      *C.SocketRxConns
	ring   *ringbuffer.Ring
	socket eal.NumaSocket
}

var _ iface.RxGroup = (*rxConns)(nil)

func (rxc *rxConns) NumaSocket() eal.NumaSocket {
	return rxc.socket
}

func (rxc *rxConns) RxGroup() (ptr unsafe.Pointer, desc string) {
	return unsafe.Pointer(rxc.c), "SocketRxConns"
}

func (rxc *rxConns) Close() {
	iface.DeactivateRxGroup(rxc)
	eal.Free(rxc.c)
	rxc.ring.Close()
	rxc.c, rxc.ring = nil, nil
}

func (rxc *rxConns) enqueue(pkt *pktmbuf.Packet) {
	if ringbuffer.Enqueue(rxc.ring, pktmbuf.Vector{pkt}) == 0 {
		pkt.Close()
	}
}

func newRxConns(ringCapacity int, socket eal.NumaSocket) (rxc *rxConns, e error) {
	rxc = &rxConns{
		socket: eal.RewriteAnyNumaSocketFirst.Rewrite(socket),
	}
	if rxc.ring, e = ringbuffer.New(4096, rxc.socket, ringbuffer.ProducerMulti, ringbuffer.ConsumerSingle); e != nil {
		return nil, e
	}

	rxc.c = eal.Zmalloc[C.SocketRxConns]("SocketRxConns", C.sizeof_SocketRxConns, rxc.socket)
	rxc.c.base.rxBurst = C.RxGroup_RxBurstFunc(C.SocketRxConns_RxBurst)
	rxc.c.ring = (*C.struct_rte_ring)(rxc.ring.Ptr())

	iface.ActivateRxGroup(rxc)
	return rxc, nil
}

// rxConns singleton.
var (
	rxConnsInstance atomic.Value
	rxConnsFaces    atomic.Int32
)

func enableRxConns(ringCapacity int, socket eal.NumaSocket) error {
	if rxConnsFaces.Inc() == 1 {
		rxc, e := newRxConns(ringCapacity, socket)
		if e != nil {
			return e
		}
		rxConnsInstance.Store(rxc)
	}
	return nil
}

func disableRxConns() {
	if rxConnsFaces.Dec() > 0 {
		return
	}
	rxc := rxConnsInstance.Swap((*rxConns)(nil)).(*rxConns)
	rxc.Close()
}
