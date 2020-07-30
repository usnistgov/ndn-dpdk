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
}

// NewFace creates a Face.
func NewFace(tr Transport) (l3face Face, e error) {
	var face l3faceImpl
	face.tr = tr
	face.rx = make(chan *ndn.Packet)
	face.tx = make(chan ndn.L3Packet)
	go face.rxLoop()
	go face.txLoop()
	return &face, nil
}

type l3faceImpl struct {
	tr Transport
	rx chan *ndn.Packet
	tx chan ndn.L3Packet
}

func (face *l3faceImpl) Transport() Transport {
	return face.tr
}

func (face *l3faceImpl) Rx() <-chan *ndn.Packet {
	return face.rx
}

func (face *l3faceImpl) Tx() chan<- ndn.L3Packet {
	return face.tx
}

func (face *l3faceImpl) rxLoop() {
	for wire := range face.tr.Rx() {
		var packet ndn.Packet
		e := tlv.Decode(wire, &packet)
		if e != nil {
			continue
		}
		face.rx <- &packet
	}
	close(face.rx)
}

func (face *l3faceImpl) txLoop() {
	transportTx := face.tr.Tx()
	for l3packet := range face.tx {
		wire, e := tlv.Encode(l3packet.ToPacket())
		if e != nil {
			continue
		}
		transportTx <- wire
	}
	close(transportTx)
}
