package ndni

/*
#include "../csrc/ndn/nack.h"
#include "../csrc/ndn/packet.h"
*/
import "C"
import (
	"fmt"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/ndn/an"
)

// Nack packet.
type Nack struct {
	m *Packet
	p *C.PNack
}

// Turn Interest into Nack.
// This overwrites the Interest.
func MakeNackFromInterest(interest *Interest, reason an.NackReason) *Nack {
	C.MakeNack(interest.m.getPtr(), C.NackReason(reason))
	return interest.m.AsNack()
}

func (nack *Nack) GetPacket() *Packet {
	return nack.m
}

func (nack *Nack) String() string {
	return fmt.Sprintf("%s~%s", nack.GetInterest(), nack.GetReason())
}

// Get *C.PNack pointer.
func (nack *Nack) GetPNackPtr() unsafe.Pointer {
	return unsafe.Pointer(nack.p)
}

func (nack *Nack) GetReason() an.NackReason {
	return an.NackReason(C.PNack_GetReason(nack.p))
}

func (nack *Nack) GetInterest() *Interest {
	return &Interest{nack.m, &nack.p.interest}
}
