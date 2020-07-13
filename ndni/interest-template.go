package ndni

/*
#include "../csrc/ndni/interest.h"
*/
import "C"
import (
	"reflect"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

// InterestTemplate is a template for Interest encoding.
// A zero InterestTemplate is invalid. It must be initialized before use.
type InterestTemplate C.InterestTemplate

// InterestTemplateFromPtr converts *C.InterestTemplate to InterestTemplate.
func InterestTemplateFromPtr(ptr unsafe.Pointer) *InterestTemplate {
	return (*InterestTemplate)(ptr)
}

func (tpl *InterestTemplate) ptr() *C.InterestTemplate {
	return (*C.InterestTemplate)(tpl)
}

// Init initializes InterestTemplate.
// Arguments should be acceptable to ndn.MakeInterest.
// Name is used as name prefix.
// Panics on error.
func (tpl *InterestTemplate) Init(args ...interface{}) {
	interest := ndn.MakeInterest(args...)
	_, wire, e := interest.MarshalTlv()
	if e != nil {
		log.WithError(e).Panic("interest.MarshalTlv error")
	}

	c := tpl.ptr()
	*c = C.InterestTemplate{}

	var prefixV, midBuf []byte
	*(*reflect.SliceHeader)(unsafe.Pointer(&prefixV)) = reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(&c.prefixV[0])),
		Len:  int(unsafe.Sizeof(c.prefixV)),
		Cap:  int(unsafe.Sizeof(c.prefixV)),
	}
	*(*reflect.SliceHeader)(unsafe.Pointer(&midBuf)) = reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(&c.midBuf[0])),
		Len:  int(unsafe.Sizeof(c.midBuf)),
		Cap:  int(unsafe.Sizeof(c.midBuf)),
	}

	d := tlv.Decoder(wire)
	for _, field := range d.Elements() {
		switch field.Type {
		case an.TtName:
			c.prefixL = C.uint16_t(copy(prefixV, field.Value))
			c.midLen = C.uint16_t(copy(midBuf, field.After))
		case an.TtNonce:
			c.nonceVOffset = c.midLen - C.uint16_t(len(field.After)+len(field.Value))
		}
	}
}

// Encode encodes an Interest via template.
func (tpl *InterestTemplate) Encode(m *pktmbuf.Packet, suffix ndn.Name, nonce uint32) *Packet {
	suffixP := NewPName(suffix)
	defer suffixP.Free()
	pktC := C.InterestTemplate_Encode(tpl.ptr(), (*C.struct_rte_mbuf)(m.Ptr()), suffixP.lname(), C.uint32_t(nonce))
	return PacketFromPtr(unsafe.Pointer(pktC))
}
