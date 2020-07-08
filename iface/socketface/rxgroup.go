package socketface

/*
#include "../../csrc/iface/rxloop.h"

uint16_t go_SocketRxGroup_RxBurst(RxGroup* rxg, struct rte_mbuf** pkts, uint16_t nPkts);
*/
import "C"
import (
	"reflect"
	"sync/atomic"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
)

var (
	nFaces  int32 // atomic
	rxQueue = make(chan *pktmbuf.Packet, 4096)
	rxgC    *C.RxGroup
)

// ChangeRxQueueCapacity changes RX queue capacity, shared among all socket faces.
// This function is non-thread-safe and can only be used when there's no active socket face.
func ChangeRxQueueCapacity(capacity int) {
	if atomic.LoadInt32(&nFaces) > 0 {
		log.Panic("cannot ChangeRxQueueCapacity with active faces")
	}
	rxQueue = make(chan *pktmbuf.Packet, capacity)
}

type rxGroup struct{}

var rxg rxGroup

func (rxGroup) IsRxGroup() {}

func (rxGroup) NumaSocket() eal.NumaSocket {
	return eal.NumaSocket{}
}

func (rxGroup) Ptr() unsafe.Pointer {
	if rxgC == nil {
		rxgC = (*C.RxGroup)(eal.Zmalloc("SocketRxGroup", C.sizeof_RxGroup, eal.NumaSocket{}))
		rxgC.rxBurstOp = C.RxGroup_RxBurst(C.go_SocketRxGroup_RxBurst)
	}
	return unsafe.Pointer(rxgC)
}

//export go_SocketRxGroup_RxBurst
func go_SocketRxGroup_RxBurst(rxg *C.RxGroup, pkts **C.struct_rte_mbuf, nPkts C.uint16_t) C.uint16_t {
	var vec []*pktmbuf.Packet
	sh := (*reflect.SliceHeader)(unsafe.Pointer(&vec))
	sh.Data = uintptr(unsafe.Pointer(pkts))
	sh.Len = int(nPkts)
	sh.Cap = sh.Len

	for i := range vec {
		select {
		case vec[i] = <-rxQueue:
		default:
			return C.uint16_t(i)
		}
	}
	return nPkts
}
