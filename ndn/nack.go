package ndn

import (
	"github.com/usnistgov/ndn-dpdk/ndn/an"
)

// Nack represents a Nack packet.
type Nack struct {
	Packet   *Packet
	Reason   uint8
	Interest Interest
}

func (nack Nack) String() string {
	return nack.Interest.String() + "~" + an.NackReasonString(nack.Reason)
}
