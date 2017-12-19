package ndn

/*
#include "lp-pkt.h"
*/
import "C"

type LpPkt struct {
	c C.LpPkt
}

type CongMark uint8

// Test whether the decoder may contain an LpPacket.
func (d *TlvDecoder) IsLpPkt() bool {
	return d.it.PeekOctet() == int(TT_LpPacket)
}

// Decode an LpPacket.
func (d *TlvDecoder) ReadLpPkt() (lpp LpPkt, e error) {
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

func (lpp *LpPkt) GetNackReason() NackReason {
	return NackReason(lpp.c.nackReason)
}

func (lpp *LpPkt) GetCongMark() CongMark {
	return CongMark(lpp.c.congMark)
}
