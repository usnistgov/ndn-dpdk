// Package endpoint implements basic consumer and producer functionality.
//
// Endpoint is the basic abstraction through which an application can communicate with the NDN network.
// It is similar to "client face" in other NDN libraries, with the enhancement that it handles these details automatically:
//   - Outgoing packets are signed and incoming packets are verified, if keys are provided.
//   - Outgoing Interests are transmitted periodically, if retransmission policy is specified.
package endpoint

import (
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
)

type lFaceL3 struct {
	*LFace
}

func (face lFaceL3) Transport() l3.Transport {
	panic("not supported")
}

func (face lFaceL3) Rx() <-chan *ndn.Packet {
	return face.ep2fw
}

func (face lFaceL3) Tx() chan<- ndn.L3Packet {
	return face.fw2ep
}

func (face lFaceL3) State() l3.TransportState {
	return l3.TransportUp
}

func (face lFaceL3) OnStateChange(cb func(st l3.TransportState)) (cancel func()) {
	panic("not supported")
}

// LFace is a logical face between endpoint (consumer or producer) and internal forwarder.
type LFace struct {
	ep2fw  chan *ndn.Packet
	fw2ep  chan ndn.L3Packet
	FwFace l3.FwFace
}

// Rx returns a channel for receiving packets from internal forwarder.
func (face *LFace) Rx() <-chan ndn.L3Packet {
	return face.fw2ep
}

// Tx returns a channel for sending packets to internal forwarder.
func (face *LFace) Tx() chan<- *ndn.Packet {
	return face.ep2fw
}

// Send attempts to send a packet to internal forwarder.
// Returns true if packet is queued, or false if packet is dropped.
func (face *LFace) Send(pkt *ndn.Packet) bool {
	select {
	case face.ep2fw <- pkt:
		return true
	default:
		return false
	}
}

// Close closes the logical face.
func (face *LFace) Close() error {
	close(face.ep2fw)
	go func() {
		n := 0
		for range face.fw2ep {
			n++
		}
	}()
	return face.FwFace.Close()
}

// NewLFace creates a logical face to an internal forwarder.
func NewLFace(fw l3.Forwarder) (face *LFace, e error) {
	if fw == nil {
		fw = l3.GetDefaultForwarder()
	}
	face = &LFace{
		ep2fw: make(chan *ndn.Packet, 16),
		fw2ep: make(chan ndn.L3Packet, 16),
	}
	l3face := lFaceL3{face}
	face.FwFace, e = fw.AddFace(l3face)
	return face, e
}
