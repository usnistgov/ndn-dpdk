package pktmbuf

/*
#include "../../csrc/dpdk/mbuf.h"
*/
import "C"
import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
)

// Vector is a vector of packet buffers.
type Vector []*Packet

func (vec Vector) ptr() **C.struct_rte_mbuf {
	return cptr.FirstPtr[*C.struct_rte_mbuf](vec)
}

// Close releases the mbufs.
func (vec Vector) Close() error {
	if len(vec) > 0 {
		C.rte_pktmbuf_free_bulk(vec.ptr(), C.uint(len(vec)))
	}
	return nil
}

// VectorFromPtr constructs Vector from **C.struct_rte_mbuf and count.
func VectorFromPtr(ptr unsafe.Pointer, count int) Vector {
	return Vector(unsafe.Slice((**Packet)(ptr), count))
}
