package ndni

/*
#include "../csrc/ndni/data.h"
*/
import "C"
import (
	"time"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

// DataGen is a Data encoder optimized for traffic generator.
type DataGen C.DataGen

// NewDataGen creates a DataGen.
func NewDataGen(m *pktmbuf.Packet, suffix ndn.Name, freshness time.Duration, content []byte) *DataGen {
	suffixP := NewPName(suffix)
	defer suffixP.Free()
	var contentV *C.uint8_t
	if len(content) > 0 {
		contentV = (*C.uint8_t)(unsafe.Pointer(&content[0]))
	}

	c := C.DataGen_New((*C.struct_rte_mbuf)(m.Ptr()), suffixP.lname(),
		C.uint32_t(freshness/time.Millisecond), C.uint16_t(len(content)), contentV)
	return (*DataGen)(c)
}

// Ptr returns *C.DataGen pointer.
func (gen *DataGen) Ptr() unsafe.Pointer {
	return unsafe.Pointer(gen)
}

func (gen *DataGen) ptr() *C.DataGen {
	return (*C.DataGen)(gen)
}

// Close discards this DataGen.
func (gen *DataGen) Close() error {
	C.DataGen_Close(gen.ptr())
	return nil
}

// Encode encodes a Data packet.
func (gen *DataGen) Encode(seg0, seg1 *pktmbuf.Packet, prefix ndn.Name) *Packet {
	prefixP := NewPName(prefix)
	defer prefixP.Free()
	pktC := C.DataGen_Encode(gen.ptr(), (*C.struct_rte_mbuf)(seg0.Ptr()), (*C.struct_rte_mbuf)(seg1.Ptr()), prefixP.lname())
	return PacketFromPtr(unsafe.Pointer(pktC))
}
