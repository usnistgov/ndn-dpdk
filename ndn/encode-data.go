package ndn

/*
#include "encode-data.h"
*/
import "C"
import (
	"fmt"
	"time"

	"ndn-dpdk/dpdk"
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
func EncodeData(m dpdk.IMbuf, namePrefix *Name, nameSuffix *Name, freshnessPeriod time.Duration, content TlvBytes) {
	C.__EncodeData((*C.struct_rte_mbuf)(m.GetPtr()),
		C.uint16_t(namePrefix.Size()), namePrefix.getValuePtr(),
		C.uint16_t(nameSuffix.Size()), nameSuffix.getValuePtr(),
		C.uint32_t(freshnessPeriod/time.Millisecond),
		C.uint16_t(len(content)), (*C.uint8_t)(content.GetPtr()))
}

// Encode a Data from flexible arguments.
// This alternate API is easier to use but less efficient.
func MakeData(m dpdk.IMbuf, name string, args ...interface{}) (*Data, error) {
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

	pkt := PacketFromDpdk(m)
	e = pkt.ParseL2()
	if e != nil {
		m.Close()
		return nil, e
	}
	e = pkt.ParseL3(dpdk.PktmbufPool{})
	if e != nil || pkt.GetL3Type() != L3PktType_Data {
		m.Close()
		return nil, e
	}
	return pkt.AsData(), nil
}
