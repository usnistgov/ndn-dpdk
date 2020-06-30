package ndn

import (
	"reflect"

	"github.com/usnistgov/ndn-dpdk/ndn/an"
)

// Nack represents a Nack packet.
type Nack struct {
	packet   *Packet
	Reason   uint8
	Interest Interest
}

// MakeNack creates a Nack from flexible arguments.
// Arguments can contain:
// - uint8 or int: set Reason
// - Interest or *Interest: set Interest, copy PitToken and CongMark
// - LpL3: copy PitToken and CongMark
func MakeNack(args ...interface{}) (nack Nack) {
	packet := Packet{Nack: &nack}
	nack.packet = &packet
	nack.Reason = an.NackUnspecified
	handleInterestArg := func(a *Interest) {
		nack.Interest = *a
		nack.Interest.packet = nil
		if ipkt := a.packet; ipkt != nil {
			packet.Lp.inheritFrom(ipkt.Lp)
		}
	}
	for _, arg := range args {
		switch a := arg.(type) {
		case uint8:
			nack.Reason = a
		case int:
			nack.Reason = uint8(a)
		case Interest:
			handleInterestArg(&a)
		case *Interest:
			handleInterestArg(a)
		case LpL3:
			packet.Lp.inheritFrom(a)
		default:
			panic("bad argument type " + reflect.TypeOf(arg).String())
		}
	}
	return nack
}

// ToPacket wraps Nack as Packet.
func (nack Nack) ToPacket() *Packet {
	if nack.packet == nil {
		packet := Packet{Nack: &nack}
		nack.packet = &packet
	}
	return nack.packet
}

// GetName returns the name of the enclosed Interest.
func (nack Nack) GetName() Name {
	return nack.Interest.Name
}

func (nack Nack) String() string {
	return nack.Interest.String() + "~" + an.NackReasonString(nack.Reason)
}
