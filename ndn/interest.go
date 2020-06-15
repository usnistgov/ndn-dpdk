package ndn

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
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand"
	"time"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
)

const (
	Interest_Headroom     = 6  // InterestTL
	Interest_SizeofGuider = 15 // Nonce(4)+InterestLifetime(4)+HopLimit(1)
	Interest_TailroomMax  = 4 + C.NAME_MAX_LENGTH + C.INTEREST_TEMPLATE_BUFLEN
)

// Interest packet.
type Interest struct {
	m *Packet
	p *C.PInterest
}

func (interest *Interest) GetPacket() *Packet {
	return interest.m
}

func (interest *Interest) String() string {
	return interest.GetName().String()
}

// Get *C.PInterest pointer.
func (interest *Interest) GetPInterestPtr() unsafe.Pointer {
	return unsafe.Pointer(interest.p)
}

func (interest *Interest) GetName() (n *Name) {
	n = new(Name)
	n.copyFromC(&interest.p.name)
	return n
}

func (interest *Interest) HasCanBePrefix() bool {
	var u C.PInterestUnpacked
	C.PInterest_Unpack(interest.p, &u)
	return bool(u.canBePrefix)
}

func (interest *Interest) HasMustBeFresh() bool {
	var u C.PInterestUnpacked
	C.PInterest_Unpack(interest.p, &u)
	return bool(u.mustBeFresh)
}

func (interest *Interest) GetNonce() uint32 {
	return uint32(interest.p.nonce)
}

func (interest *Interest) GetLifetime() time.Duration {
	return time.Duration(interest.p.lifetime) * time.Millisecond
}

func (interest *Interest) GetHopLimit() uint8 {
	return uint8(interest.p.hopLimit)
}

func (interest *Interest) GetFhs() (fhs []*Name) {
	var u C.PInterestUnpacked
	C.PInterest_Unpack(interest.p, &u)
	fhs = make([]*Name, int(u.nFhs))
	for i := range fhs {
		nameL := interest.p.fhNameL[i]
		nameV := unsafe.Pointer(interest.p.fhNameV[i])
		fhs[i], _ = NewName(TlvBytes(C.GoBytes(nameV, C.int(nameL))))
	}
	return fhs
}

func (interest *Interest) GetActiveFhIndex() int {
	var u C.PInterestUnpacked
	C.PInterest_Unpack(interest.p, &u)
	return int(u.activeFh)
}

func (interest *Interest) SelectActiveFh(index int) error {
	var u C.PInterestUnpacked
	C.PInterest_Unpack(interest.p, &u)
	if index < -1 || index >= int(u.nFhs) {
		return errors.New("fhindex out of range")
	}

	e := C.PInterest_SelectActiveFh(interest.p, C.int8_t(index))
	if e != C.NdnError_OK {
		return NdnError(e)
	}
	return nil
}

// Modify Interest guiders.
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

type ActiveFHDelegation int

func InterestTemplateFromPtr(ptr unsafe.Pointer) *InterestTemplate {
	return (*InterestTemplate)(ptr)
}

// Initialize InterestTemplate from flexible arguments.
// Increase mbuf headroom with InterestMbufExtraHeadroom.
// Specify Name with string or *Name.
// Specify CanBePrefix with `CanBePrefixFlag`.
// Specify MustBeFresh with `MustBeFreshFlag`.
// Specify ForwardingHint with FHDelegation (repeatable).
// Specify InterestLifetime with time.Duration.
// Specify HopLimit with uint8.
// ApplicationParameters and Signature are not supported.
func (tpl *InterestTemplate) Init(args ...interface{}) (e error) {
	cbp := false
	mbf := false
	var fh TlvBytes
	lifetime := uint32(C.DEFAULT_INTEREST_LIFETIME)
	hopLimit := uint8(0xFF)

	for i := 0; i < len(args); i++ {
		switch a := args[i].(type) {
		case string:
			if name, e := ParseName(a); e != nil {
				return e
			} else {
				tpl.PrefixL = uint16(copy(tpl.PrefixV[:], ([]byte)(name.GetValue())))
			}
		case *Name:
			tpl.PrefixL = uint16(copy(tpl.PrefixV[:], ([]byte)(a.GetValue())))
		case tCanBePrefix:
			cbp = true
		case tMustBeFresh:
			mbf = true
		case FHDelegation:
			if name, e := ParseName(a.Name); e != nil {
				return e
			} else {
				prefV := make([]byte, 4)
				binary.BigEndian.PutUint32(prefV, uint32(a.Preference))
				fh = fh.Join(EncodeTlv(TT_Delegation, EncodeTlv(TT_Preference, TlvBytes(prefV)), name.Encode()))
			}
		case time.Duration:
			lifetime = uint32(a / time.Millisecond)
		case uint8:
			hopLimit = a
		default:
			return fmt.Errorf("unrecognized argument type %T", a)
		}
	}

	var mid TlvBytes
	if cbp {
		mid = mid.Join(EncodeTlv(TT_CanBePrefix))
	}
	if mbf {
		mid = mid.Join(EncodeTlv(TT_MustBeFresh))
	}
	if len(fh) > 0 {
		mid = mid.Join(EncodeTlv(TT_ForwardingHint, fh))
	}
	{
		nonceV := make(TlvBytes, 4)
		mid = mid.Join(EncodeTlv(TT_Nonce, nonceV))
		tpl.NonceOff = uint16(len(mid) - 4)
	}
	if lifetime != C.DEFAULT_INTEREST_LIFETIME {
		lifetimeV := make([]byte, 4)
		binary.BigEndian.PutUint32(lifetimeV, lifetime)
		mid = mid.Join(EncodeTlv(TT_InterestLifetime, TlvBytes(lifetimeV)))
	}
	if hopLimit != 0xFF {
		mid = mid.Join(EncodeTlv(TT_HopLimit, TlvBytes{hopLimit}))
	}
	tpl.MidLen = uint16(copy(tpl.MidBuf[:], ([]byte)(mid)))
	return nil
}

// Encode an Interest from template.
// must be empty and is the only segment.
// mbuf headroom should be at least Interest_Headroom plus Ethernet and NDNLP headers.
// mbuf tailroom should fit the whole packet; a safe value is Interest_TailroomMax.
func (tpl *InterestTemplate) Encode(m *pktmbuf.Packet, suffix *Name, nonce uint32) {
	var suffixV TlvBytes
	if suffix != nil {
		suffixV = suffix.GetValue()
	}

	C.EncodeInterest_((*C.struct_rte_mbuf)(m.GetPtr()), (*C.InterestTemplate)(unsafe.Pointer(tpl)),
		C.uint16_t(len(suffixV)), (*C.uint8_t)(suffixV.GetPtr()), C.uint32_t(nonce))
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
