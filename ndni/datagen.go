package ndni

/*
#include "../csrc/ndni/data.h"
*/
import "C"
import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

// DataGen is a Data encoder optimized for traffic generator.
type DataGen C.DataGen

// NewDataGen creates a DataGen.
// m is a pktmbuf with at least DataGenBufLen + len(content) buffer size.
// It can be allocated from PayloadMempool.
// Arguments should be acceptable to ndn.MakeData.
// Name is used as name suffix.
// Panics on error.
func NewDataGen(m *pktmbuf.Packet, args ...interface{}) *DataGen {
	data := ndn.MakeData(args...)
	_, wire, e := data.MarshalTlv()
	if e != nil {
		log.WithError(e).Panic("data.MarshalTlv error")
	}

	var nameL, tplSize int
	d := tlv.Decoder(wire)
DecodeLoop:
	for _, field := range d.Elements() {
		switch field.Type {
		case an.TtName:
			nameL = field.Length()
			tplSize = nameL + len(field.After)
			break DecodeLoop
		}
	}

	m.SetHeadroom(0)
	if e := m.Append(wire[len(wire)-tplSize:]); e != nil {
		log.WithError(e).Panic("mbuf.Append error")
	}

	(*C.struct_rte_mbuf)(m.Ptr()).vlan_tci = C.uint16_t(nameL)
	return (*DataGen)(m.Ptr())
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
	return pktmbuf.PacketFromPtr(unsafe.Pointer(gen)).Close()
}

// Encode encodes a Data packet.
func (gen *DataGen) Encode(seg0, seg1 *pktmbuf.Packet, prefix ndn.Name) *Packet {
	prefixP := NewPName(prefix)
	defer prefixP.Free()
	pktC := C.DataGen_Encode(gen.ptr(), (*C.struct_rte_mbuf)(seg0.Ptr()), (*C.struct_rte_mbuf)(seg1.Ptr()), prefixP.lname())
	return PacketFromPtr(unsafe.Pointer(pktC))
}
