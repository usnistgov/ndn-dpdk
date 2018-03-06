package ndn

/*
#include "lp.h"
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/dpdk"
)

type LpL3 struct {
	c C.LpL3
}

func (l3 *LpL3) GetPitToken() uint64 {
	return uint64(l3.c.pitToken)
}

func (l3 *LpL3) GetNackReason() NackReason {
	return NackReason(l3.c.nackReason)
}

type CongMark uint8

func (l3 *LpL3) GetCongMark() CongMark {
	return CongMark(l3.c.congMark)
}

type LpHeader struct {
	LpL3
	l2 C.LpL2
}

func (lph *LpHeader) GetFragFields() (seqNo uint64, fragIndex uint16, fragCount uint16) {
	return uint64(lph.l2.seqNo), uint16(lph.l2.fragIndex), uint16(lph.l2.fragCount)
}

func PrependLpHeader_GetHeadroom() int {
	return int(C.PrependLpHeader_GetHeadroom())
}

func (lph *LpHeader) Prepend(pkt dpdk.IMbuf, payloadL int) {
	lphC := (*C.LpHeader)(unsafe.Pointer(lph))
	C.PrependLpHeader((*C.struct_rte_mbuf)(pkt.GetPtr()), lphC, C.uint32_t(payloadL))
}

func init() {
	var lph LpHeader
	if unsafe.Sizeof(lph) != C.sizeof_LpHeader {
		panic("ndn.LpHeader definition does not match C.LpHeader")
	}
}
