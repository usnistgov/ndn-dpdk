package ndn

/*
#include "interest-pkt.h"
*/
import "C"
import (
	"time"
	"unsafe"

	"ndn-dpdk/dpdk"
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

// Template to make an Interest.
type InterestTemplate struct {
	c          C.InterestTemplate
	NamePrefix TlvBytes
	NameSuffix TlvBytes
	FwHints    TlvBytes
}

func NewInterestTemplate() (tpl *InterestTemplate) {
	tpl = new(InterestTemplate)
	tpl.c.lifetime = C.DEFAULT_INTEREST_LIFETIME
	return tpl
}

func (tpl *InterestTemplate) SetNamePrefixFromUri(uri string) error {
	prefix, e := EncodeNameComponentsFromUri(uri)
	if e != nil {
		return e
	}
	tpl.NamePrefix = prefix
	return nil
}

func (tpl *InterestTemplate) GetMustBeFresh() bool {
	return bool(tpl.c.mustBeFresh)
}

func (tpl *InterestTemplate) SetMustBeFresh(v bool) {
	tpl.c.mustBeFresh = C.bool(v)
}

func (tpl *InterestTemplate) GetInterestLifetime() time.Duration {
	return time.Duration(tpl.c.lifetime) * time.Millisecond
}

func (tpl *InterestTemplate) SetInterestLifetime(lifetime time.Duration) {
	tpl.c.lifetime = C.uint32_t(lifetime / time.Millisecond)
}

func (tpl *InterestTemplate) EncodeTo(m dpdk.IMbuf) {
	tpl.c.namePrefixSize = C.uint16_t(len(tpl.NamePrefix))
	tpl.c.nameSuffixSize = C.uint16_t(len(tpl.NameSuffix))
	tpl.c.fwHintsSize = C.uint16_t(len(tpl.FwHints))
	C.__EncodeInterest((*C.struct_rte_mbuf)(m.GetPtr()), &tpl.c,
		(*C.uint8_t)(tpl.NamePrefix.GetPtr()), (*C.uint8_t)(tpl.NameSuffix.GetPtr()),
		(*C.uint8_t)(tpl.FwHints.GetPtr()))
}
