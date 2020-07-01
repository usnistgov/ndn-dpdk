package ndni

/*
#include "../csrc/ndn/interest.h"

typedef struct PInterestUnpacked
{
	bool canBePrefix;
	bool mustBeFresh;
	uint8_t nFhs;
	int8_t activeFh;
} PInterestUnpacked;

static void
PInterest_Unpack(const PInterest* p, PInterestUnpacked* u)
{
	u->canBePrefix = p->canBePrefix;
	u->mustBeFresh = p->mustBeFresh;
	u->nFhs = p->nFhs;
	u->activeFh = p->activeFh;
}
*/
import "C"
import (
	"errors"
	"time"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

func (pinterest *pInterest) ptr() *C.PInterest {
	return (*C.PInterest)(unsafe.Pointer(pinterest))
}

func (pinterest *pInterest) unpack() (u C.PInterestUnpacked) {
	C.PInterest_Unpack(pinterest.ptr(), &u)
	return u
}

// Interest represents an Interest packet in mbuf.
type Interest struct {
	m *Packet
	p *pInterest
}

// AsPacket converts Interest to Packet.
func (interest Interest) AsPacket() *Packet {
	return interest.m
}

// ToNInterest copies this packet into ndn.Data.
// Panics on error.
func (interest Interest) ToNInterest() ndn.Interest {
	return *interest.m.ToNPacket().Interest
}

func (interest Interest) String() string {
	return interest.Name().String()
}

// PInterestPtr returns *C.PInterest pointer.
func (interest Interest) PInterestPtr() unsafe.Pointer {
	return unsafe.Pointer(interest.p)
}

// Name returns Interest name.
func (interest Interest) Name() ndn.Name {
	return interest.p.Name.ToName()
}

// HasCanBePrefix returns CanBePrefix flag.
func (interest Interest) HasCanBePrefix() bool {
	return bool(interest.p.unpack().canBePrefix)
}

// HasMustBeFresh returns MustBeFresh flag.
func (interest Interest) HasMustBeFresh() bool {
	return bool(interest.p.unpack().mustBeFresh)
}

// Nonce returns Nonce.
func (interest Interest) Nonce() ndn.Nonce {
	return ndn.NonceFromUint(interest.p.Nonce)
}

// Lifetime returns InterestLifetime.
func (interest Interest) Lifetime() time.Duration {
	return time.Duration(interest.p.Lifetime) * time.Millisecond
}

// HopLimit returns HopLimit.
func (interest Interest) HopLimit() uint8 {
	return uint8(interest.p.HopLimit)
}

// FwHints returns a list of forwarding hints.
func (interest Interest) FwHints() (fhs []ndn.Name) {
	fhs = make([]ndn.Name, int(interest.p.unpack().nFhs))
	for i := range fhs {
		var lname LName
		lname.Value = interest.p.FhNameV[i]
		lname.Length = interest.p.FhNameL[i]
		fhs[i] = lname.ToName()
	}
	return fhs
}

// ActiveFwHintIndex returns active forwarding hint index.
func (interest Interest) ActiveFwHintIndex() int {
	return int(interest.p.unpack().activeFh)
}

// SelectActiveFh sets active forwarding hint index.
func (interest *Interest) SelectActiveFh(index int) error {
	if index < -1 || index >= int(interest.p.unpack().nFhs) {
		return errors.New("fhindex out of range")
	}

	e := C.PInterest_SelectActiveFh(interest.p.ptr(), C.int8_t(index))
	if e != C.NdnErrOK {
		return NdnError(e)
	}
	return nil
}

// Modify updates Interest guiders.
func (interest *Interest) Modify(nonce uint32, lifetime time.Duration,
	hopLimit uint8, headerMp, guiderMp, indirectMp *pktmbuf.Pool) *Interest {
	outPktC := C.ModifyInterest(interest.m.ptr(), C.uint32_t(nonce),
		C.uint32_t(lifetime/time.Millisecond), C.uint8_t(hopLimit),
		(*C.struct_rte_mempool)(headerMp.Ptr()),
		(*C.struct_rte_mempool)(guiderMp.Ptr()),
		(*C.struct_rte_mempool)(indirectMp.Ptr()))
	if outPktC == nil {
		return nil
	}
	return PacketFromPtr(unsafe.Pointer(outPktC)).AsInterest()
}

// InterestTemplateFromPtr converts *C.InterestTemplate to InterestTemplate.
func InterestTemplateFromPtr(ptr unsafe.Pointer) *InterestTemplate {
	return (*InterestTemplate)(ptr)
}

// Init initializes InterestTemplate.
// Arguments should be acceptable to ndn.MakeInterest.
// Name is used as name prefix.
func (tpl *InterestTemplate) Init(args ...interface{}) error {
	interest := ndn.MakeInterest(args...)
	_, wire, e := interest.MarshalTlv()
	if e != nil {
		return e
	}

	d := tlv.Decoder(wire)
	for _, field := range d.Elements() {
		switch field.Type {
		case an.TtName:
			tpl.PrefixL = uint16(copy(tpl.PrefixV[:], field.Value))
			tpl.MidLen = uint16(copy(tpl.MidBuf[:], field.After))
		case an.TtNonce:
			tpl.NonceOff = tpl.MidLen - uint16(len(field.After)+len(field.Value))
		}
	}
	return nil
}

// Encode encodes an Interest via template.
// mbuf must be empty and is the only segment.
// mbuf headroom should be at least InterestEstimatedHeadroom plus Ethernet and NDNLP headers.
// mbuf tailroom should fit the whole packet; a safe value is InterestEstimatedTailroom.
func (tpl *InterestTemplate) Encode(m *pktmbuf.Packet, suffix ndn.Name, nonce uint32) {
	var suffixV []byte
	if len(suffix) > 0 {
		suffixV, _ = suffix.MarshalBinary()
	}

	C.EncodeInterest_((*C.struct_rte_mbuf)(m.Ptr()), (*C.InterestTemplate)(unsafe.Pointer(tpl)),
		C.uint16_t(len(suffixV)), bytesToPtr(suffixV), C.uint32_t(nonce))
}
