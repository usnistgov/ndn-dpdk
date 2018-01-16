package dpdk

/*
#include "mbuf.h"
*/
import "C"
import (
	"unsafe"
)

type IMbuf interface {
	Close()
	GetPtr() unsafe.Pointer
	iMbufFlag()
}

type Mbuf struct {
	ptr *C.struct_rte_mbuf
	// DO NOT add other fields: *Mbuf is passed to C code as rte_mbuf**
}

// Construct Mbuf from native *C.struct_rte_mbuf pointer.
func MbufFromPtr(ptr unsafe.Pointer) Mbuf {
	return Mbuf{(*C.struct_rte_mbuf)(ptr)}
}

func (m Mbuf) iMbufFlag() {
	panic("Mbuf.isMbufFlag should not be invoked")
}

// Get native *C.struct_rte_mbuf pointer to use in other packages.
func (m Mbuf) GetPtr() unsafe.Pointer {
	return unsafe.Pointer(m.ptr)
}

func (m Mbuf) IsValid() bool {
	return m.ptr != nil
}

func (m Mbuf) Close() error {
	C.rte_pktmbuf_free(m.ptr)
	m.ptr = nil
	return nil
}

func (m Mbuf) AsPacket() Packet {
	return Packet{m}
}
