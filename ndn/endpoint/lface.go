// Package endpoint implements basic consumer and producer functionality.
//
// Endpoint is the basic abstraction through which an application can communicate with the NDN network.
// It is similar to "client face" in other NDN libraries, with the enhancement that it handles these details automatically:
//  - Outgoing packets are signed and incoming packets are verified, if keys are provided.
//  - Outgoing Interests are transmitted periodically, if retransmission policy is specified.
package endpoint

import (
	"io"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
)

// lFace is a logical face between endpoint (consumer or producer) and internal forwarder.
type lFace struct {
	ep2fw  chan *ndn.Packet
	fw2ep  chan ndn.L3Packet
	fwFace l3.FwFace
}

func (face *lFace) Transport() l3.Transport {
	panic("not supported")
}

func (face *lFace) Rx() <-chan *ndn.Packet {
	return face.ep2fw
}

func (face *lFace) Tx() chan<- ndn.L3Packet {
	return face.fw2ep
}

func (face *lFace) State() l3.TransportState {
	return l3.TransportUp
}

func (face *lFace) OnStateChange(cb func(st l3.TransportState)) io.Closer {
	panic("not supported")
}

func (face *lFace) Close() error {
	close(face.ep2fw)
	go func() {
		n := 0
		for range face.fw2ep {
			n++
		}
	}()
	return face.fwFace.Close()
}

func newLFace(fw l3.Forwarder) (face *lFace, e error) {
	face = &lFace{
		ep2fw: make(chan *ndn.Packet, 16),
		fw2ep: make(chan ndn.L3Packet, 16),
	}
	face.fwFace, e = fw.AddFace(face)
	return face, e
}
