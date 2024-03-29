package ndn

import (
	"reflect"

	"github.com/usnistgov/ndn-dpdk/ndn/an"
)

// Nack represents a Nack packet.
//
// Nack struct does not support encoding or decoding. Instead, you can encode nack.ToPacket(),
// or decode as Packet then access the Nack.
type Nack struct {
	packet   *Packet
	Reason   uint8
	Interest Interest
}

var (
	_ L3Packet = Nack{}
)

// MakeNack creates a Nack from flexible arguments.
// Arguments can contain:
//   - uint8 or int: set Reason
//   - Interest or *Interest: set Interest, copy PitToken and CongMark
//   - LpL3: copy PitToken and CongMark
func MakeNack(args ...any) (nack Nack) {
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
func (nack Nack) ToPacket() (packet *Packet) {
	packet = &Packet{}
	if nack.packet != nil {
		*packet = *nack.packet
	}
	packet.Nack = &nack
	return packet
}

// Name returns the name of the enclosed Interest.
func (nack Nack) Name() Name {
	return nack.Interest.Name
}

func (nack Nack) String() string {
	return nack.Interest.String() + "~" + an.NackReasonString(nack.Reason)
}
