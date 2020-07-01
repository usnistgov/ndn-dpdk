package ndni

/*
#include "../csrc/ndn/nack.h"
#include "../csrc/ndn/packet.h"
*/
import "C"
import (
	"fmt"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
)

func (pnack *pNack) ptr() *C.PNack {
	return (*C.PNack)(unsafe.Pointer(pnack))
}

// Nack packet.
type Nack struct {
	m *Packet
	p *pNack
}

// MakeNackFromInterest turns an Interest into a Nack.
// This overwrites the Interest.
func MakeNackFromInterest(interest *Interest, reason uint8) *Nack {
	C.MakeNack(interest.m.ptr(), C.NackReason(reason))
	return interest.m.AsNack()
}

// AsPacket converts Nack to Packet.
func (nack Nack) AsPacket() *Packet {
	return nack.m
}

// ToNNack copies this packet into ndn.Nack.
// Panics on error.
func (nack Nack) ToNNack() ndn.Nack {
	return *nack.m.ToNPacket().Nack
}

func (nack Nack) String() string {
	return fmt.Sprintf("%s~%s", nack.Interest(), an.NackReasonString(nack.Reason()))
}

// PNackPtr returns *C.PNack pointer.
func (nack Nack) PNackPtr() unsafe.Pointer {
	return unsafe.Pointer(nack.p)
}

// Reason returns Nack reason.
func (nack Nack) Reason() uint8 {
	return nack.p.Lpl3.NackReason
}

// Interest returns the Interest enclosed in Nack.
func (nack Nack) Interest() *Interest {
	return &Interest{nack.m, &nack.p.Interest}
}
