package ndni

/*
#include "../csrc/ndn/lp.h"
*/
import "C"
import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
)

func PrependLpHeader_GetHeadroom() int {
	return int(C.PrependLpHeader_GetHeadroom())
}

func (lph *LpHeader) Prepend(pkt *pktmbuf.Packet, payloadL int) {
	lphC := (*C.LpHeader)(unsafe.Pointer(lph))
	C.PrependLpHeader((*C.struct_rte_mbuf)(pkt.GetPtr()), lphC, C.uint32_t(payloadL))
}

func init() {
	var lph LpHeader
	if unsafe.Sizeof(lph) != C.sizeof_LpHeader {
		panic("ndni.LpHeader definition does not match C.LpHeader")
	}
}
