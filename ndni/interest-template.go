package ndni

/*
#include "../csrc/ndni/interest.h"
*/
import "C"
import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/core/nnduration"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
	"go.uber.org/zap"
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
	wire, e := tlv.EncodeValueOnly(interest)
	if e != nil {
		logger.Panic("encode Interest error", zap.Error(e))
	}

	c := tpl.ptr()
	*c = C.InterestTemplate{}

	d := tlv.DecodingBuffer(wire)
	for _, de := range d.Elements() {
		switch de.Type {
		case an.TtName:
			c.prefixL = C.uint16_t(copy(cptr.AsByteSlice(&c.prefixV), de.Value))
			c.midLen = C.uint16_t(copy(cptr.AsByteSlice(&c.midBuf), de.After))
		case an.TtNonce:
			c.nonceVOffset = c.midLen - C.uint16_t(len(de.After)+len(de.Value))
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

// InterestTemplateConfig is a JSON serializable object that can construct InterestTemplate.
type InterestTemplateConfig struct {
	Prefix           ndn.Name                `json:"prefix"`
	CanBePrefix      bool                    `json:"canBePrefix,omitempty"`
	MustBeFresh      bool                    `json:"mustBeFresh,omitempty"`
	InterestLifetime nnduration.Milliseconds `json:"interestLifetime,omitempty"`
	HopLimit         ndn.HopLimit            `json:"hopLimit,omitempty"`
}

// Apply initializes InterestTemplate.
func (cfg InterestTemplateConfig) Apply(tpl *InterestTemplate) {
	a := []interface{}{cfg.Prefix}
	if cfg.CanBePrefix {
		a = append(a, ndn.CanBePrefixFlag)
	}
	if cfg.MustBeFresh {
		a = append(a, ndn.MustBeFreshFlag)
	}
	if lifetime := cfg.InterestLifetime.Duration(); lifetime != 0 {
		a = append(a, lifetime)
	}
	if cfg.HopLimit != 0 {
		a = append(a, cfg.HopLimit)
	}
	tpl.Init(a...)
}
