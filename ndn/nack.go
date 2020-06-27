package ndn

import (
	"reflect"

	"github.com/usnistgov/ndn-dpdk/ndn/an"
)

// Nack represents a Nack packet.
type Nack struct {
	Packet   *Packet
	Reason   uint8
	Interest Interest
}

// MakeNack creates a Nack from flexible arguments.
// Arguments can contain:
// - uint8 or int: set Reason
// - Interest: set Interest, copy PitToken and CongMark
// - LpHeader: copy PitToken and CongMark
func MakeNack(args ...interface{}) (nack Nack) {
	packet := Packet{Nack: &nack}
	nack.Packet = &packet
	nack.Reason = an.NackUnspecified
	for _, arg := range args {
		switch a := arg.(type) {
		case uint8:
			nack.Reason = a
		case int:
			nack.Reason = uint8(a)
		case Interest:
			nack.Interest = a
			nack.Interest.Packet = nil
			if ipkt := a.Packet; ipkt != nil {
				packet.Lp.inheritFrom(ipkt.Lp)
			}
		case LpHeader:
			packet.Lp.inheritFrom(a)
		default:
			panic("bad argument type " + reflect.TypeOf(arg).String())
		}
	}
	return nack
}

// GetName returns the name of the enclosed Interest.
func (nack Nack) GetName() Name {
	return nack.Interest.Name
}

func (nack Nack) String() string {
	return nack.Interest.String() + "~" + an.NackReasonString(nack.Reason)
}
