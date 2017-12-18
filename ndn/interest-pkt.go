package ndn

/*
#include "interest-pkt.h"
*/
import "C"
import (
	"time"
	"unsafe"
)

type InterestPkt struct {
	c C.InterestPkt
}

// Test whether the decoder may contain an Interest.
func (d *TlvDecoder) IsInterest() bool {
	return d.it.PeekOctet() == int(TT_Interest)
}

// Decode an Interest.
func (d *TlvDecoder) ReadInterest() (interest InterestPkt, e error) {
	res := C.DecodeInterest(d.getPtr(), &interest.c)
	if res != C.NdnError_OK {
		return InterestPkt{}, NdnError(res)
	}
	return interest, nil
}

func (interest *InterestPkt) GetName() *Name {
	return (*Name)(unsafe.Pointer(&interest.c.name))
}

func (interest *InterestPkt) HasMustBeFresh() bool {
	return bool(interest.c.mustBeFresh)
}

func (interest *InterestPkt) GetNonce() uint32 {
	return uint32(C.InterestPkt_GetNonce(&interest.c))
}

func (interest *InterestPkt) SetNonce(nonce uint32) {
	C.InterestPkt_SetNonce(&interest.c, C.uint32_t(nonce))
}

func (interest *InterestPkt) GetLifetime() time.Duration {
	return time.Duration(interest.c.lifetime) * time.Millisecond
}

func (interest *InterestPkt) GetFwHints() []*Name {
	fhs := make([]*Name, int(interest.c.nFwHints))
	for i := range fhs {
		fhs[i] = (*Name)(unsafe.Pointer(&interest.c.fwHints[i]))
	}
	return fhs
}
