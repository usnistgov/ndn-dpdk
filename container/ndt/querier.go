package ndt

/*
#include "../../csrc/ndt/ndt.h"

enum {
	c_NdtQuerier_offsetof_nHits = offsetof(NdtQuerier, nHits)
};
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

// Close releases memory.
func (ndq *Querier) Close() error {
	eal.Free(ndq)
	return nil
}

// Lookup queries a name and increments hit counters.
func (ndq *Querier) Lookup(name ndn.Name) uint8 {
	nameP := ndni.NewPName(name)
	defer nameP.Free()
	return uint8(C.NdtQuerier_Lookup(ndq.ptr(), (*C.PName)(nameP.Ptr())))
}

func (ndq *Querier) hitCounters(nEntries int) (hits []uint32) {
	return unsafe.Slice((*uint32)(unsafe.Add(unsafe.Pointer(ndq), C.c_NdtQuerier_offsetof_nHits)), nEntries)
}

func newQuerier(ndt *Ndt, socket eal.NumaSocket) *Querier {
	return (*Querier)(C.NdtQuerier_New(ndt.replicas[socket].ptr(), C.int(socket.ID())))
}
