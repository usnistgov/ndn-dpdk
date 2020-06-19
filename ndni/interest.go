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

const (
	Interest_Headroom     = 6  // InterestTL
	Interest_SizeofGuider = 15 // Nonce(4)+InterestLifetime(4)+HopLimit(1)
	Interest_TailroomMax  = 4 + C.NAME_MAX_LENGTH + C.INTEREST_TEMPLATE_BUFLEN
)

func (pinterest *pInterest) getPtr() *C.PInterest {
	return (*C.PInterest)(unsafe.Pointer(pinterest))
}

func (pinterest *pInterest) unpack() (u C.PInterestUnpacked) {
	C.PInterest_Unpack(pinterest.getPtr(), &u)
	return u
}

// Interest represents an Interest packet in mbuf.
type Interest struct {
	m *Packet
	p *pInterest
}

// GetPacket converts Interest to Packet.
func (interest Interest) GetPacket() *Packet {
	return interest.m
}

func (interest Interest) String() string {
	return interest.GetName().String()
}

// GetPInterestPtr returns *C.PInterest pointer.
func (interest Interest) GetPInterestPtr() unsafe.Pointer {
	return unsafe.Pointer(interest.p)
}

// GetName returns Interest name.
func (interest Interest) GetName() ndn.Name {
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

// GetNonce returns Nonce.
func (interest Interest) GetNonce() ndn.Nonce {
	return ndn.NonceFromUint(interest.p.Nonce)
}

// GetLifetime returns InterestLifetime.
func (interest Interest) GetLifetime() time.Duration {
	return time.Duration(interest.p.Lifetime) * time.Millisecond
}

// GetHopLimit returns HopLimit.
func (interest Interest) GetHopLimit() uint8 {
	return uint8(interest.p.HopLimit)
}

// GetFhs returns a list of forwarding hints.
func (interest Interest) GetFhs() (fhs []ndn.Name) {
	fhs = make([]ndn.Name, int(interest.p.unpack().nFhs))
	for i := range fhs {
		var lname LName
		lname.Value = interest.p.FhNameV[i]
		lname.Length = interest.p.FhNameL[i]
		fhs[i] = lname.ToName()
	}
	return fhs
}

// GetActiveFhIndex returns active forwarding hint index.
func (interest Interest) GetActiveFhIndex() int {
	return int(interest.p.unpack().activeFh)
}

// SelectActiveFh sets active forwarding hint index.
func (interest *Interest) SelectActiveFh(index int) error {
	if index < -1 || index >= int(interest.p.unpack().nFhs) {
		return errors.New("fhindex out of range")
	}

	e := C.PInterest_SelectActiveFh(interest.p.getPtr(), C.int8_t(index))
	if e != C.NdnErrOK {
		return NdnError(e)
	}
	return nil
}

// Modify updates Interest guiders.
// headerMp element size should be at least Interest_Headroom plus Ethernet and NDNLP headers.
// guiderMp element size should be at least Interest_SizeofGuider.
func (interest *Interest) Modify(nonce uint32, lifetime time.Duration,
	hopLimit uint8, headerMp, guiderMp, indirectMp *pktmbuf.Pool) *Interest {
	outPktC := C.ModifyInterest(interest.m.getPtr(), C.uint32_t(nonce),
		C.uint32_t(lifetime/time.Millisecond), C.uint8_t(hopLimit),
		(*C.struct_rte_mempool)(headerMp.GetPtr()),
		(*C.struct_rte_mempool)(guiderMp.GetPtr()),
		(*C.struct_rte_mempool)(indirectMp.GetPtr()))
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
		switch an.TlvType(field.Type) {
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
// mbuf headroom should be at least Interest_Headroom plus Ethernet and NDNLP headers.
// mbuf tailroom should fit the whole packet; a safe value is Interest_TailroomMax.
func (tpl *InterestTemplate) Encode(m *pktmbuf.Packet, suffix ndn.Name, nonce uint32) {
	var suffixV []byte
	if len(suffix) > 0 {
		suffixV, _ = suffix.MarshalBinary()
	}

	C.EncodeInterest_((*C.struct_rte_mbuf)(m.GetPtr()), (*C.InterestTemplate)(unsafe.Pointer(tpl)),
		C.uint16_t(len(suffixV)), bytesToPtr(suffixV), C.uint32_t(nonce))
}
