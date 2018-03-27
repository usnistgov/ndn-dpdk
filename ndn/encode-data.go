package ndn

/*
#include "encode-data.h"
*/
import "C"
import (
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
func EncodeData(m dpdk.IMbuf, name *Name, freshnessPeriod time.Duration, content TlvBytes) {
	C.__EncodeData((*C.struct_rte_mbuf)(m.GetPtr()),
		C.uint16_t(name.Size()), name.getValuePtr(),
		C.uint32_t(freshnessPeriod/time.Millisecond),
		C.uint16_t(len(content)), (*C.uint8_t)(content.GetPtr()))
}
