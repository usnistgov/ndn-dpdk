package dpdk

/*
#include "mbuf.h"
*/
import "C"
import (
	"unsafe"
)

const MBUF_DEFAULT_HEADROOM = C.RTE_PKTMBUF_HEADROOM

type IMbuf interface {
	Close() error
	GetPtr() unsafe.Pointer
	iMbufFlag()
}

type Mbuf struct {
	c *C.struct_rte_mbuf
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
	return unsafe.Pointer(m.c)
}

func (m Mbuf) IsValid() bool {
	return m.c != nil
}

func (m Mbuf) Close() error {
	C.rte_pktmbuf_free(m.c)
	return nil
}

func (m Mbuf) AsPacket() Packet {
	return Packet{m}
}

func init() {
	var m Mbuf
	if unsafe.Sizeof(m) != unsafe.Sizeof(m.c) {
		panic("sizeof dpdk.Mbuf differs from *C.struct_rte_mbuf")
	}
}
