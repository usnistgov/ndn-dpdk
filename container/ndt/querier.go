package ndt

/*
#include "../../csrc/ndt/ndt.h"

uint32_t* c_NdtQuerier_Hits(NdtQuerier* ndq) { return ndq->nHits; }
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
	return unsafe.Pointer(ndq.ptr())
}

func (ndq *Querier) ptr() *C.NdtQuerier {
	return (*C.NdtQuerier)(ndq)
}

// Close releases memory.
func (ndq *Querier) Close() error {
	eal.Free(ndq.ptr())
	return nil
}

// Lookup queries a name and increments hit counters.
func (ndq *Querier) Lookup(name ndn.Name) uint8 {
	nameP := ndni.NewPName(name)
	defer nameP.Free()
	return uint8(C.NdtQuerier_Lookup(ndq.ptr(), (*C.PName)(nameP.Ptr())))
}

func (ndq *Querier) hitCounters(nEntries int) (hits []uint32) {
	return unsafe.Slice((*uint32)(unsafe.Pointer(C.c_NdtQuerier_Hits(ndq.ptr()))), nEntries)
}

func newThread(ndt *Ndt, socket eal.NumaSocket) *Querier {
	c := C.NdtQuerier_New(ndt.replicas[socket].ptr(), C.int(socket.ID()))
	return (*Querier)(c)
}
