package ndni

/*
#include "../csrc/ndn/encode-data.h"
*/
import "C"
import (
	"time"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

// Encode a Data.
func EncodeData(pkt *pktmbuf.Packet, prefix, suffix ndn.Name, freshnessPeriod time.Duration, content []byte) {
	prefixV, _ := prefix.MarshalBinary()
	suffixV, _ := suffix.MarshalBinary()
	C.EncodeData_((*C.struct_rte_mbuf)(pkt.GetPtr()),
		C.uint16_t(len(prefixV)), bytesToPtr(prefixV),
		C.uint16_t(len(suffixV)), bytesToPtr(suffixV),
		C.uint32_t(freshnessPeriod/time.Millisecond),
		C.uint16_t(len(content)), bytesToPtr(content))
}

type DataGen struct {
	c *C.DataGen
}

func NewDataGen(m *pktmbuf.Packet, suffix ndn.Name, freshnessPeriod time.Duration, content []byte) (gen DataGen) {
	suffixV, _ := suffix.MarshalBinary()
	gen.c = C.MakeDataGen_((*C.struct_rte_mbuf)(m.GetPtr()),
		C.uint16_t(len(suffixV)), bytesToPtr(suffixV),
		C.uint32_t(freshnessPeriod/time.Millisecond),
		C.uint16_t(len(content)), bytesToPtr(content))
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

func (gen DataGen) Encode(seg0, seg1 *pktmbuf.Packet, prefix ndn.Name) {
	prefixV, _ := prefix.MarshalBinary()
	C.DataGen_Encode_(gen.c,
		(*C.struct_rte_mbuf)(seg0.GetPtr()), (*C.struct_rte_mbuf)(seg1.GetPtr()),
		C.uint16_t(len(prefixV)), bytesToPtr(prefixV))
}
