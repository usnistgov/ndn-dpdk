package ndn

/*
#include "encode-interest.h"
*/
import "C"
import (
	"encoding/binary"
	"fmt"
	"math/rand"
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
	fh         TlvBytes
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

func (tpl *InterestTemplate) AppendFH(preference int, name *Name) {
	prefV := make([]byte, 4)
	binary.BigEndian.PutUint32(prefV, uint32(preference))
	prefTLV := EncodeTlv(TT_Preference, TlvBytes(prefV))

	delV := JoinTlvBytes(prefTLV, name.Encode())
	delTLV := EncodeTlv(TT_Delegation, delV)

	tpl.fh = append(tpl.fh, delTLV...)
	tpl.c.fhL = C.uint16_t(len(tpl.fh))
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
	C.__InterestTemplate_Prepare(&tpl.c, (*C.uint8_t)(tpl.buffer.GetPtr()), size, (*C.uint8_t)(tpl.fh.GetPtr()))
}

// Encode an Interest from template.
func (tpl *InterestTemplate) Encode(m dpdk.IMbuf, nameSuffix *Name, nonce uint32, paramV TlvBytes) {
	tpl.prepare()

	var nameSuffixV TlvBytes
	if nameSuffix != nil {
		nameSuffixV = nameSuffix.GetValue()
	}
	C.__EncodeInterest((*C.struct_rte_mbuf)(m.GetPtr()), &tpl.c, (*C.uint8_t)(tpl.buffer.GetPtr()),
		C.uint16_t(len(nameSuffixV)), (*C.uint8_t)(nameSuffixV.GetPtr()),
		C.uint32_t(nonce), C.uint16_t(len(paramV)), (*C.uint8_t)(paramV.GetPtr()),
		(*C.uint8_t)(tpl.namePrefix.GetPtr()))
}

type tCanBePrefix bool
type tMustBeFresh bool

const (
	CanBePrefixFlag = tCanBePrefix(true)
	MustBeFreshFlag = tMustBeFresh(true)
)

type FHDelegation struct {
	Preference int
	Name       string
}

// Encode an Interest from flexible arguments.
// This alternate API is easier to use but less efficient.
func MakeInterest(m dpdk.IMbuf, name string, args ...interface{}) (interest *Interest, e error) {
	var n *Name
	nonce := rand.Uint32()
	var param TlvBytes
	tpl := NewInterestTemplate()

	if n, e = ParseName(name); e != nil {
		m.Close()
		return nil, e
	}

	for i := 0; i < len(args); i++ {
		switch a := args[i].(type) {
		case tCanBePrefix:
			tpl.SetCanBePrefix(true)
		case tMustBeFresh:
			tpl.SetMustBeFresh(true)
		case FHDelegation:
			var fhName *Name
			if fhName, e = ParseName(a.Name); e != nil {
				m.Close()
				return nil, e
			}
			tpl.AppendFH(a.Preference, fhName)
		case uint32:
			nonce = a
		case time.Duration:
			tpl.SetInterestLifetime(a)
		case HopLimit:
			tpl.SetHopLimit(a)
		case TlvBytes:
			param = a
		default:
			m.Close()
			return nil, fmt.Errorf("unrecognized argument type %T", a)
		}
	}

	tpl.Encode(m, n, nonce, param)

	pkt := PacketFromDpdk(m)
	if e = pkt.ParseL2(); e != nil {
		m.Close()
		return nil, e
	}
	if e = pkt.ParseL3(dpdk.PktmbufPool{}); e != nil || pkt.GetL3Type() != L3PktType_Interest {
		m.Close()
		return nil, e
	}
	return pkt.AsInterest(), nil
}
