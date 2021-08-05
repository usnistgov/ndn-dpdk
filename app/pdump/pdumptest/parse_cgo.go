package pdumptest

/*
#include "../../../csrc/pdump/parse.h"
*/
import "C"
import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf/mbuftestenv"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

func extractName(npkt tlv.Fielder) []byte {
	wire, _ := tlv.EncodeFrom(npkt)
	m := mbuftestenv.MakePacket(wire)
	defer m.Close()
	lnameC := C.Pdump_ExtractName((*C.struct_rte_mbuf)(m.Ptr()))
	return C.GoBytes(unsafe.Pointer(lnameC.value), C.int(lnameC.length))
}
