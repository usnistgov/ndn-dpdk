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

func (pnack *pNack) getPtr() *C.PNack {
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
	C.MakeNack(interest.m.getPtr(), C.NackReason(reason))
	return interest.m.AsNack()
}

// GetPacket converts Nack to Packet.
func (nack Nack) GetPacket() *Packet {
	return nack.m
}

// ToNNack copies this packet into ndn.Nack.
// Panics on error.
func (nack Nack) ToNNack() ndn.Nack {
	return *nack.m.ToNPacket().Nack
}

func (nack Nack) String() string {
	return fmt.Sprintf("%s~%s", nack.GetInterest(), an.NackReasonString(nack.GetReason()))
}

// GetPNackPtr returns *C.PNack pointer.
func (nack Nack) GetPNackPtr() unsafe.Pointer {
	return unsafe.Pointer(nack.p)
}

// GetReason returns Nack reason.
func (nack Nack) GetReason() uint8 {
	return nack.p.Lpl3.NackReason
}

// GetInterest returns the Interest enclosed in Nack.
func (nack Nack) GetInterest() *Interest {
	return &Interest{nack.m, &nack.p.Interest}
}
