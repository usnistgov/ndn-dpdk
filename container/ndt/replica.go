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

func (ndtr *replica) Close() error {
	eal.Free(ndtr.ptr())
	return nil
}

func (ndtr *replica) Read(i int) Entry {
	return Entry{
		Index: i,
		Value: int(uint8(C.Ndt_Read(ndtr.ptr(), C.uint64_t(i)))),
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

func newReplica(cfg Config, socket eal.NumaSocket) *replica {
	c := C.Ndt_New_(C.uint64_t(cfg.Capacity), C.int(socket.ID()))
	c.sampleMask = C.uint64_t(cfg.SampleInterval)
	c.prefixLen = C.uint16_t(cfg.PrefixLen)
	return (*replica)(c)
}
