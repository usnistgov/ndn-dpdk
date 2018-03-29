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

func (interest *Interest) GetHopLimit() uint8 {
	return uint8(interest.p.hopLimit)
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

func (interest *Interest) MatchesData(data *Data) bool {
	return bool(C.PInterest_MatchesData(interest.p, data.m.c))
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
