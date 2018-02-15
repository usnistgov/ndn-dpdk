package ndn

/*
#include "lp.h"
*/
import "C"
import "ndn-dpdk/dpdk"

type CongMark uint8

type LpHeader struct {
	c C.LpHeader
}

func (lph *LpHeader) GetFragFields() (seqNo uint64, fragIndex uint16, fragCount uint16) {
	return uint64(lph.c.l2.seqNo), uint16(lph.c.l2.fragIndex), uint16(lph.c.l2.fragCount)
}

func (lph *LpHeader) GetPitToken() uint64 {
	return uint64(lph.c.l3.pitToken)
}

func (lph *LpHeader) GetNackReason() NackReason {
	return NackReason(lph.c.l3.nackReason)
}

func (lph *LpHeader) GetCongMark() CongMark {
	return CongMark(lph.c.l3.congMark)
}

func EncodeLpHeader_GetHeadroom() int {
	return int(C.EncodeLpHeader_GetHeadroom())
}

func EncodeLpHeader_GetTailroom() int {
	return int(C.EncodeLpHeader_GetTailroom())
}

func (lph *LpHeader) Encode(pkt dpdk.IMbuf, payloadL int) {
	C.EncodeLpHeader((*C.struct_rte_mbuf)(pkt.GetPtr()), &lph.c, C.uint32_t(payloadL))
}
