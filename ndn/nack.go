package ndn

/*
#include "nack.h"
#include "packet.h"
*/
import "C"
import (
	"fmt"
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

func (nr NackReason) String() string {
	switch nr {
	case NackReason_Congestion:
		return "Congestion"
	case NackReason_Duplicate:
		return "Duplicate"
	case NackReason_NoRoute:
		return "NoRoute"
	}
	return fmt.Sprintf("%d", nr)
}

// Nack packet.
type Nack struct {
	m Packet
	p *C.PNack
}

func (nack *Nack) GetReason() NackReason {
	return NackReason(C.PNack_GetReason(nack.p))
}

func (nack *Nack) GetInterest() *Interest {
	return &Interest{nack.m, &nack.p.interest}
}
