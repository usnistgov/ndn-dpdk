package ndni

/*
#include "../csrc/ndn/nack.h"
#include "../csrc/ndn/packet.h"
*/
import "C"
import (
	"fmt"
	"strconv"
	"unsafe"
)

// Indicate a Nack reason.
type NackReason uint8

const (
	NackReason_None        = NackReason(C.NackReason_None)
	NackReason_Congestion  = NackReason(C.NackReason_Congestion)
	NackReason_Duplicate   = NackReason(C.NackReason_Duplicate)
	NackReason_NoRoute     = NackReason(C.NackReason_NoRoute)
	NackReason_Unspecified = NackReason(C.NackReason_Unspecified)
)

func (reason NackReason) String() string {
	switch reason {
	case NackReason_Congestion:
		return "Congestion"
	case NackReason_Duplicate:
		return "Duplicate"
	case NackReason_NoRoute:
		return "NoRoute"
	}
	return strconv.Itoa(int(reason))
}

func ParseNackReason(s string) NackReason {
	switch s {
	case "Congestion":
		return NackReason_Congestion
	case "Duplicate":
		return NackReason_Duplicate
	case "NoRoute":
		return NackReason_NoRoute
	}
	return NackReason_Unspecified
}

// Nack packet.
type Nack struct {
	m *Packet
	p *C.PNack
}

// Turn Interest into Nack.
// This overwrites the Interest.
func MakeNackFromInterest(interest *Interest, reason NackReason) *Nack {
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

func (nack *Nack) GetReason() NackReason {
	return NackReason(C.PNack_GetReason(nack.p))
}

func (nack *Nack) GetInterest() *Interest {
	return &Interest{nack.m, &nack.p.interest}
}
