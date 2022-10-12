package ndt

/*
#include "../../csrc/ndt/ndt.h"
*/
import "C"
import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

// Querier represents an NDT querier with counters.
type Querier C.NdtQuerier

// Ptr returns *C.NdtQuerier pointer.
func (ndq *Querier) Ptr() unsafe.Pointer {
	return unsafe.Pointer(ndq)
}

func (ndq *Querier) ptr() *C.NdtQuerier {
	return (*C.NdtQuerier)(ndq)
}

// Init initializes the Querier.
func (ndq *Querier) Init(ndt *Ndt, socket eal.NumaSocket) {
	*ndq = Querier{
		ndt:   ndt.getReplica(socket).ptr(),
		nHits: eal.ZmallocAligned[C.uint32_t]("NdtQuerier.nHits", C.sizeof_uint32_t*ndt.cfg.Capacity, 1, socket),
	}
	ndt.queriers.Put(ndq)
}

// Clear releases memory allocated in Init.
func (ndq *Querier) Clear(ndt *Ndt) {
	ndt.queriers.Remove(ndq)
	if ndq.nHits != nil {
		eal.Free(ndq.nHits)
	}
	*ndq = Querier{}
}

// Lookup queries a name and increments hit counters.
func (ndq *Querier) Lookup(name ndn.Name) uint8 {
	nameP := ndni.NewPName(name)
	defer nameP.Free()
	return uint8(C.NdtQuerier_Lookup(ndq.ptr(), (*C.PName)(nameP.Ptr())))
}

func (ndq *Querier) hitCounters(nEntries int) (hits []uint32) {
	return unsafe.Slice((*uint32)(unsafe.Pointer(ndq.nHits)), nEntries)
}
