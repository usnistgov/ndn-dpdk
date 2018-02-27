package ndn

/*
#include "interest.h"
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

func (interest *Interest) GetFhIndex() int {
	return int(interest.p.thisFhIndex)
}

func (interest *Interest) SetFhIndex(index int) error {
	if index < -1 || index >= int(interest.p.nFhs) {
		return errors.New("fhindex out of range")
	}
	if index == -1 {
		interest.p.thisFhIndex = -1
		return nil
	}

	e := C.PInterest_ParseFh(interest.p, C.uint8_t(index))
	if e != C.NdnError_OK {
		return NdnError(e)
	}
	return nil
}

func ModifyInterest_SizeofGuider() int {
	return int(C.ModifyInterest_SizeofGuider())
}

type InterestMod struct {
	Nonce    uint32
	Lifetime time.Duration
	HopLimit HopLimit
}

func (interest *Interest) Modify(mod InterestMod, header dpdk.IMbuf, guider dpdk.IMbuf,
	indirectMp dpdk.PktmbufPool) *Interest {
	var modC C.InterestMod
	modC.nonce = C.uint32_t(mod.Nonce)
	modC.lifetime = C.uint32_t(mod.Lifetime / time.Millisecond)
	modC.hopLimit = C.HopLimit(mod.HopLimit)

	outPktC := C.ModifyInterest(interest.m.c, &modC, (*C.struct_rte_mbuf)(header.GetPtr()),
		(*C.struct_rte_mbuf)(guider.GetPtr()), (*C.struct_rte_mempool)(indirectMp.GetPtr()))
	if outPktC == nil {
		return nil
	}
	return Packet{outPktC}.AsInterest()
}
