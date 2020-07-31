// Package l3 defines a network layer face abstraction.
//
// The Transport interface defines a lower layer communication channel.
// It knows NDN-TLV structure, but not NDN packet types.
// It should be implemented for different communication technologies.
// NDN-DPDK codebase offers Transport implementations for Unix, UDP, TCP, and AF_PACKET sockets.
//
// The Face type is the service exposed to the network layer.
// It allows sending and receiving packets on a Transport.
package l3

import (
	"io"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

// Face represents a communicate channel to send and receive TLV packets.
type Face interface {
	// Transport returns the underlying transport.
	Transport() Transport

	// Rx returns a channel to receive incoming packets.
	// This function always returns the same channel.
	// This channel is closed when the face is closed.
	Rx() <-chan *ndn.Packet

	// Tx returns a channel to send outgoing packets.
	// This function always returns the same channel.
	// Closing this channel causes the face to close.
	Tx() chan<- ndn.L3Packet

	State() TransportState
	OnStateChange(cb func(st TransportState)) io.Closer
}

// NewFace creates a Face.
func NewFace(tr Transport) (Face, error) {
	f := &face{
		tr: tr,
		rx: make(chan *ndn.Packet),
		tx: make(chan ndn.L3Packet),
	}
	go f.rxLoop()
	go f.txLoop()
	return f, nil
}

type face struct {
	tr Transport
	rx chan *ndn.Packet
	tx chan ndn.L3Packet
}

func (f *face) Transport() Transport {
	return f.tr
}

func (f *face) Rx() <-chan *ndn.Packet {
	return f.rx
}

func (f *face) Tx() chan<- ndn.L3Packet {
	return f.tx
}

func (f *face) State() TransportState {
	return f.tr.State()
}

func (f *face) OnStateChange(cb func(st TransportState)) io.Closer {
	return f.tr.OnStateChange(cb)
}

func (f *face) rxLoop() {
	for wire := range f.tr.Rx() {
		var packet ndn.Packet
		e := tlv.Decode(wire, &packet)
		if e != nil {
			continue
		}
		f.rx <- &packet
	}
	close(f.rx)
}

func (f *face) txLoop() {
	transportTx := f.tr.Tx()
	for l3packet := range f.tx {
		wire, e := tlv.Encode(l3packet.ToPacket())
		if e != nil {
			continue
		}
		transportTx <- wire
	}
	close(transportTx)
}
