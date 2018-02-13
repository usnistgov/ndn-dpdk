package ndn

/*
#include "interest.h"
*/
import "C"
import (
	"time"
	"unsafe"

	"ndn-dpdk/dpdk"
)

// Interest packet.
type Interest struct {
	m Packet
	p *C.PInterest
}

func (interest *Interest) GetName() (n *Name) {
	n = new(Name)
	n.copyFromC(&interest.p.name)
	return n
}

func (interest *Interest) HasCanBePrefix() bool {
	return bool(interest.p.canBePrefix)
}

func (interest *Interest) HasMustBeFresh() bool {
	return bool(interest.p.mustBeFresh)
}

func (interest *Interest) GetNonce() uint32 {
	return uint32(interest.p.nonce)
}

func (interest *Interest) GetLifetime() time.Duration {
	return time.Duration(interest.p.lifetime) * time.Millisecond
}

// Interest HopLimit field.
type HopLimit uint16

const (
	HOP_LIMIT_OMITTED HopLimit = 0x0100 // HopLimit is omitted.
	HOP_LIMIT_ZERO    HopLimit = 0x0101 // HopLimit was zero before decrementing.
)

func (interest *Interest) GetHopLimit() HopLimit {
	return HopLimit(interest.p.hopLimit)
}

func (interest *Interest) GetFhs() (fhs []*Name) {
	fhs = make([]*Name, int(interest.p.nFhs))
	for i := range fhs {
		lname := interest.p.fh[i]
		fhs[i], _ = NewName(TlvBytes(C.GoBytes(unsafe.Pointer(lname.value), C.int(lname.length))))
	}
	return fhs
}

func EncodeInterest_GetHeadroom() int {
	return int(C.EncodeInterest_GetHeadroom())
}

func EncodeInterest_GetTailroomMax() int {
	return int(C.EncodeInterest_GetTailroomMax())
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
	prefix, e := ParseName(uri)
	if e != nil {
		return e
	}
	tpl.NamePrefix = prefix.GetValue()
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
