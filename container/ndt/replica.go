package ndt

/*
#include "../../csrc/ndt/ndt.h"
*/
import "C"
import (
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

type replica C.Ndt

func (ndtr *replica) ptr() *C.Ndt {
	return (*C.Ndt)(ndtr)
}

func (ndtr *replica) Read(i uint64) Entry {
	return Entry{
		Index: i,
		Value: uint8(C.Ndt_Read(ndtr.ptr(), C.uint64_t(i))),
	}
}

func (ndtr *replica) Update(index uint64, value uint8) {
	C.Ndt_Update(ndtr.ptr(), C.uint64_t(index), C.uint8_t(value))
}

func (ndtr *replica) Lookup(name ndn.Name) (index uint64, value uint8) {
	nameP := ndni.NewPName(name)
	defer nameP.Free()
	var indexC C.uint64_t
	value = uint8(C.Ndt_Lookup(ndtr.ptr(), (*C.PName)(nameP.Ptr()), &indexC))
	return uint64(indexC), value
}

func newReplica(cfg Config, socket eal.NumaSocket) (ndtr *replica) {
	ndtr = (*replica)(C.Ndt_New(C.uint64_t(cfg.Capacity), C.int(socket.ID())))
	ndtr.sampleMask = C.uint64_t(cfg.SampleInterval - 1)
	ndtr.prefixLen = C.uint16_t(cfg.PrefixLen)
	return ndtr
}
