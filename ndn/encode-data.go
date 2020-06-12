package ndn

/*
#include "encode-data.h"
*/
import "C"
import (
	"fmt"
	"time"
	"unsafe"

	"ndn-dpdk/dpdk/pktmbuf"
)

func EncodeData_GetHeadroom() int {
	return int(C.EncodeData_GetHeadroom())
}

func EncodeData_GetTailroom(nameL int, contentL int) int {
	return int(C.EncodeData_GetTailroom(C.uint16_t(nameL), C.uint16_t(contentL)))
}

func EncodeData_GetTailroomMax() int {
	return int(C.EncodeData_GetTailroomMax())
}

// Encode a Data.
func EncodeData(pkt *pktmbuf.Packet, namePrefix *Name, nameSuffix *Name, freshnessPeriod time.Duration, content TlvBytes) {
	C.EncodeData_((*C.struct_rte_mbuf)(pkt.GetPtr()),
		C.uint16_t(namePrefix.Size()), namePrefix.getValuePtr(),
		C.uint16_t(nameSuffix.Size()), nameSuffix.getValuePtr(),
		C.uint32_t(freshnessPeriod/time.Millisecond),
		C.uint16_t(len(content)), (*C.uint8_t)(content.GetPtr()))
}

// Encode a Data from flexible arguments.
// This alternate API is easier to use but less efficient.
func MakeData(m *pktmbuf.Packet, name string, args ...interface{}) (*Data, error) {
	n, e := ParseName(name)
	if e != nil {
		m.Close()
		return nil, e
	}
	var freshnessPeriod time.Duration
	var content TlvBytes

	for _, arg := range args {
		switch a := arg.(type) {
		case time.Duration:
			freshnessPeriod = a
		case TlvBytes:
			content = a
		default:
			m.Close()
			return nil, fmt.Errorf("unrecognized argument type %T", a)
		}
	}

	EncodeData(m, n, nil, freshnessPeriod, content)

	pkt := PacketFromMbuf(m)
	e = pkt.ParseL2()
	if e != nil {
		m.Close()
		return nil, e
	}
	e = pkt.ParseL3(nil)
	if e != nil || pkt.GetL3Type() != L3PktType_Data {
		m.Close()
		return nil, e
	}
	return pkt.AsData(), nil
}

func DataGen_GetHeadroom0() int {
	return int(C.EncodeData_GetHeadroom())
}

func DataGen_GetTailroom0(namePrefixL int) int {
	return int(C.DataGen_GetTailroom0(C.uint16_t(namePrefixL)))
}

func DataGen_GetTailroom1(nameSuffixL, contentL int) int {
	return int(C.DataGen_GetTailroom1(C.uint16_t(nameSuffixL), C.uint16_t(contentL)))
}

type DataGen struct {
	c *C.DataGen
}

func NewDataGen(m *pktmbuf.Packet, nameSuffix *Name, freshnessPeriod time.Duration, content TlvBytes) (gen DataGen) {
	gen.c = C.MakeDataGen_((*C.struct_rte_mbuf)(m.GetPtr()),
		C.uint16_t(nameSuffix.Size()), nameSuffix.getValuePtr(),
		C.uint32_t(freshnessPeriod/time.Millisecond),
		C.uint16_t(len(content)), (*C.uint8_t)(content.GetPtr()))
	return gen
}

func DataGenFromPtr(ptr unsafe.Pointer) (gen DataGen) {
	return DataGen{(*C.DataGen)(ptr)}
}

func (gen DataGen) GetPtr() unsafe.Pointer {
	return unsafe.Pointer(gen.c)
}

func (gen DataGen) Close() error {
	C.DataGen_Close(gen.c)
	return nil
}

func (gen DataGen) Encode(seg0, seg1 *pktmbuf.Packet, namePrefix *Name) {
	C.DataGen_Encode_(gen.c,
		(*C.struct_rte_mbuf)(seg0.GetPtr()), (*C.struct_rte_mbuf)(seg1.GetPtr()),
		C.uint16_t(namePrefix.Size()), namePrefix.getValuePtr())
}
