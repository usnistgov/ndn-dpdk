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
	"bytes"
	"errors"
	"fmt"
	"math/rand"
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

// GetNonce returns Nonce in native endianness.
func (interest Interest) GetNonce() uint32 {
	return uint32(interest.p.Nonce)
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

func (del FHDelegation) Encode() []byte {
	wire, _ := tlv.EncodeElement(an.TtDelegation,
		tlv.MakeElementNNI(an.TtPreference, del.Preference),
		ndn.ParseName(del.Name),
	)
	return wire
}

type ActiveFHDelegation int

func InterestTemplateFromPtr(ptr unsafe.Pointer) *InterestTemplate {
	return (*InterestTemplate)(ptr)
}

// Initialize InterestTemplate from flexible arguments.
// Increase mbuf headroom with InterestMbufExtraHeadroom.
// Specify Name with string or ndn.Name.
// Specify CanBePrefix with `CanBePrefixFlag`.
// Specify MustBeFresh with `MustBeFreshFlag`.
// Specify ForwardingHint with FHDelegation (repeatable).
// Specify InterestLifetime with time.Duration.
// Specify HopLimit with uint8.
// ApplicationParameters and Signature are not supported.
func (tpl *InterestTemplate) Init(args ...interface{}) (e error) {
	cbp := false
	mbf := false
	var fh [][]byte
	lifetime := uint32(C.DEFAULT_INTEREST_LIFETIME)
	hopLimit := uint8(0xFF)

	for i := 0; i < len(args); i++ {
		switch a := args[i].(type) {
		case string:
			name := ndn.ParseName(a)
			nameV, _ := name.MarshalBinary()
			tpl.PrefixL = uint16(copy(tpl.PrefixV[:], nameV))
		case ndn.Name:
			nameV, _ := a.MarshalBinary()
			tpl.PrefixL = uint16(copy(tpl.PrefixV[:], nameV))
		case tCanBePrefix:
			cbp = true
		case tMustBeFresh:
			mbf = true
		case FHDelegation:
			fh = append(fh, a.Encode())
		case time.Duration:
			lifetime = uint32(a / time.Millisecond)
		case uint8:
			hopLimit = a
		default:
			return fmt.Errorf("unrecognized argument type %T", a)
		}
	}

	var mid []tlv.Element
	if cbp {
		mid = append(mid, tlv.MakeElement(an.TtCanBePrefix, nil))
	}
	if mbf {
		mid = append(mid, tlv.MakeElement(an.TtMustBeFresh, nil))
	}
	if len(fh) > 0 {
		mid = append(mid, tlv.MakeElement(an.TtForwardingHint, bytes.Join(fh, nil)))
	}
	{
		mid = append(mid, tlv.MakeElement(an.TtNonce, make([]byte, 4)))
		wire, _ := tlv.EncodeValue(mid)
		tpl.NonceOff = uint16(len(wire) - 4)
	}
	if lifetime != C.DEFAULT_INTEREST_LIFETIME {
		mid = append(mid, tlv.MakeElementNNI(an.TtInterestLifetime, lifetime))
	}
	if hopLimit != 0xFF {
		mid = append(mid, tlv.MakeElement(an.TtHopLimit, []byte{hopLimit}))
	}
	wire, _ := tlv.EncodeValue(mid)
	tpl.MidLen = uint16(copy(tpl.MidBuf[:], wire))
	return nil
}

// Encode an Interest from template.
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

// Encode an Interest from flexible arguments.
// In addition to argument types supported by `func (tpl *InterestTemplate) Init`:
// Specify Nonce with uint32.
// Choose active ForwardingHint delegation with ActiveFHDelegation.
func MakeInterest(m *pktmbuf.Packet, args ...interface{}) (interest *Interest, e error) {
	nonce := rand.Uint32()
	activeFh := -1
	var tplArgs []interface{}

	for _, arg := range args {
		switch a := arg.(type) {
		case ActiveFHDelegation:
			activeFh = int(a)
		case uint32:
			nonce = a
		default:
			tplArgs = append(tplArgs, arg)
		}
	}

	var tpl InterestTemplate
	if e = tpl.Init(tplArgs...); e != nil {
		m.Close()
		return nil, e
	}

	tpl.Encode(m, nil, nonce)
	pkt := PacketFromMbuf(m)
	if e = pkt.ParseL2(); e != nil {
		m.Close()
		return nil, e
	}
	if e = pkt.ParseL3(nil); e != nil || pkt.GetL3Type() != L3PktType_Interest {
		m.Close()
		return nil, e
	}

	interest = pkt.AsInterest()
	if activeFh >= 0 {
		if e = interest.SelectActiveFh(activeFh); e != nil {
			m.Close()
			return nil, e
		}
	}
	return interest, nil
}
