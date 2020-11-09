package pktmbuf

/*
#include "../../csrc/dpdk/mbuf.h"
*/
import "C"
import (
	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"unsafe"
)

// Vector is a vector of packet buffers.
type Vector []*Packet

// Ptr returns **C.struct_rte_mbuf pointer.
func (vec Vector) Ptr() unsafe.Pointer {
	ptr, _ := cptr.ParseCptrArray(vec)
	return ptr
}

func (vec Vector) ptr() **C.struct_rte_mbuf {
	return (**C.struct_rte_mbuf)(vec.Ptr())
}

// Close releases the mbufs.
func (vec Vector) Close() error {
	if len(vec) == 0 {
		return nil
	}
	C.rte_pktmbuf_free_bulk(vec.ptr(), C.uint(len(vec)))
	return nil
}
