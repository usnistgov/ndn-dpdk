package ndn

/*
#include "interest.h"

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

func ModifyInterest_SizeofGuider() int {
	return int(C.ModifyInterest_SizeofGuider())
}

func (interest *Interest) Modify(nonce uint32, lifetime time.Duration,
	hopLimit uint8, headerMp dpdk.PktmbufPool,
	guiderMp dpdk.PktmbufPool, indirectMp dpdk.PktmbufPool) *Interest {
	outPktC := C.ModifyInterest(interest.m.c, C.uint32_t(nonce),
		C.uint32_t(lifetime/time.Millisecond), C.uint8_t(hopLimit),
		(*C.struct_rte_mempool)(headerMp.GetPtr()),
		(*C.struct_rte_mempool)(guiderMp.GetPtr()),
		(*C.struct_rte_mempool)(indirectMp.GetPtr()))
	if outPktC == nil {
		return nil
	}
	return Packet{outPktC}.AsInterest()
}
