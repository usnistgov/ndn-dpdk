package ndn

import (
	"github.com/usnistgov/ndn-dpdk/ndn/an"
)

// Nack represents a Nack packet.
type Nack struct {
	Packet   *Packet
	Reason   an.NackReason
	Interest Interest
}

func (nack Nack) String() string {
	return nack.Interest.String() + "~" + nack.Reason.String()
}
