package ndn

/*
#include "encode-interest.h"
*/
import "C"
import (
	"time"

	"ndn-dpdk/dpdk"
)

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
