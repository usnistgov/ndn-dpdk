package pktmbuf

/*
#include "../../csrc/dpdk/mbuf.h"
*/
import "C"
import (
	"ndn-dpdk/core/cptr"
	"unsafe"
)

// Vector is a vector of packet buffers.
type Vector []*Packet

// GetPtr returns **C.struct_rte_mbuf pointer.
func (vec Vector) GetPtr() unsafe.Pointer {
	ptr, _ := cptr.ParseCptrArray(vec)
	return ptr
}

func (vec Vector) getPtr() **C.struct_rte_mbuf {
	return (**C.struct_rte_mbuf)(vec.GetPtr())
}

// Close releases the mbufs.
func (vec Vector) Close() error {
	if len(vec) == 0 {
		return nil
	}
	C.rte_pktmbuf_free_bulk_(vec.getPtr(), C.uint(len(vec)))
	return nil
}
