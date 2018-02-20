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

func (interest *Interest) GetPacket() Packet {
	return interest.m
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
	HOP_LIMIT_OMITTED = HopLimit(C.HOP_LIMIT_OMITTED) // HopLimit is omitted.
	HOP_LIMIT_ZERO    = HopLimit(C.HOP_LIMIT_ZERO)    // HopLimit was zero before decrementing.
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

// Template to encode an Interest.
type InterestTemplate struct {
	c          C.InterestTemplate
	buffer     TlvBytes
	namePrefix TlvBytes
}

func NewInterestTemplate() (tpl *InterestTemplate) {
	tpl = new(InterestTemplate)
	tpl.c.lifetime = C.DEFAULT_INTEREST_LIFETIME
	tpl.c.hopLimit = C.HOP_LIMIT_OMITTED
	return tpl
}

func (tpl *InterestTemplate) SetNamePrefix(v *Name) {
	tpl.namePrefix = v.GetValue()
	tpl.c.namePrefix.length = C.uint16_t(len(tpl.namePrefix))
}

func (tpl *InterestTemplate) SetCanBePrefix(v bool) {
	tpl.buffer = nil
	tpl.c.canBePrefix = C.bool(v)
}

func (tpl *InterestTemplate) SetMustBeFresh(v bool) {
	tpl.buffer = nil
	tpl.c.mustBeFresh = C.bool(v)
}

func (tpl *InterestTemplate) SetInterestLifetime(v time.Duration) {
	tpl.buffer = nil
	tpl.c.lifetime = C.uint32_t(v / time.Millisecond)
}

func (tpl *InterestTemplate) SetHopLimit(v HopLimit) {
	tpl.buffer = nil
	tpl.c.hopLimit = C.HopLimit(v)
}

func (tpl *InterestTemplate) prepare() {
	if tpl.buffer != nil {
		return
	}
	size := C.__InterestTemplate_Prepare(&tpl.c, nil, 0, nil)
	tpl.buffer = make(TlvBytes, int(size))
	C.__InterestTemplate_Prepare(&tpl.c, (*C.uint8_t)(tpl.buffer.GetPtr()), size, nil)
}

func (tpl *InterestTemplate) Encode(m dpdk.IMbuf, nameSuffix *Name, paramV TlvBytes) {
	tpl.prepare()

	var nameSuffixV TlvBytes
	if nameSuffix != nil {
		nameSuffixV = nameSuffix.GetValue()
	}
	C.__EncodeInterest((*C.struct_rte_mbuf)(m.GetPtr()), &tpl.c, (*C.uint8_t)(tpl.buffer.GetPtr()),
		C.uint16_t(len(nameSuffixV)), (*C.uint8_t)(nameSuffixV.GetPtr()),
		C.uint16_t(len(paramV)), (*C.uint8_t)(paramV.GetPtr()),
		(*C.uint8_t)(tpl.namePrefix.GetPtr()))
}
