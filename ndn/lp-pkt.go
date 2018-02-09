package ndn

/*
#include "lp-pkt.h"
*/
import "C"
import "ndn-dpdk/dpdk"

type LpPkt struct {
	c C.LpPkt
}

type CongMark uint8

// Test whether the decoder may contain an LpPacket.
func (d *TlvDecodePos) IsLpPkt() bool {
	return d.it.PeekOctet() == int(TT_LpPacket)
}

// Decode an LpPacket.
func (d *TlvDecodePos) ReadLpPkt() (lpp LpPkt, e error) {
	res := C.DecodeLpPkt(d.getPtr(), &lpp.c)
	if res != C.NdnError_OK {
		return LpPkt{}, NdnError(res)
	}
	return lpp, nil
}

func (lpp *LpPkt) HasPayload() bool {
	return bool(C.LpPkt_HasPayload(&lpp.c))
}

func (lpp *LpPkt) IsFragmented() bool {
	return bool(C.LpPkt_IsFragmented(&lpp.c))
}

func (lpp *LpPkt) GetFragFields() (seqNo uint64, fragIndex uint16, fragCount uint16) {
	return uint64(lpp.c.seqNo), uint16(lpp.c.fragIndex), uint16(lpp.c.fragCount)
}

func (lpp *LpPkt) GetPitToken() uint64 {
	return uint64(lpp.c.pitToken)
}

func (lpp *LpPkt) GetNackReason() NackReason {
	return NackReason(lpp.c.nackReason)
}

func (lpp *LpPkt) GetCongMark() CongMark {
	return CongMark(lpp.c.congMark)
}

func EncodeLpHeaders_GetHeadroom() int {
	return int(C.EncodeLpHeaders_GetHeadroom())
}

func EncodeLpHeaders_GetTailroom() int {
	return int(C.EncodeLpHeaders_GetTailroom())
}

func (lpp *LpPkt) EncodeHeaders(pkt dpdk.IMbuf) {
	C.EncodeLpHeaders((*C.struct_rte_mbuf)(pkt.GetPtr()), &lpp.c)
}
