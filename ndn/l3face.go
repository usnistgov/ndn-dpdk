package ndn

import (
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

// Transport represents a communicate channel to send and receive TLV packets.
type Transport interface {
	// GetRx returns a channel to receive incoming TLV elements.
	// This function always returns the same channel.
	// This channel is closed when the transport is closed.
	GetRx() <-chan []byte

	// GetTx returns a channel to send outgoing TLV elements.
	// This function always returns the same channel.
	// Closing this channel causes the transport to close.
	GetTx() chan<- []byte
}

// L3Face represents a communicate channel to send and receive TLV packets.
type L3Face interface {
	// GetTransport returns the underlying transport.
	GetTransport() Transport

	// GetRx returns a channel to receive incoming packets.
	// This function always returns the same channel.
	// This channel is closed when the face is closed.
	GetRx() <-chan *Packet

	// GetTx returns a channel to send outgoing packets.
	// This function always returns the same channel.
	// Closing this channel causes the face to close.
	GetTx() chan<- L3Packet
}

// NewL3Face creates an L3Face.
func NewL3Face(tr Transport) (l3face L3Face, e error) {
	var face l3faceImpl
	face.tr = tr
	face.rx = make(chan *Packet)
	face.tx = make(chan L3Packet)
	go face.rxLoop()
	go face.txLoop()
	return &face, nil
}

type l3faceImpl struct {
	tr Transport
	rx chan *Packet
	tx chan L3Packet
}

func (face *l3faceImpl) GetTransport() Transport {
	return face.tr
}

func (face *l3faceImpl) GetRx() <-chan *Packet {
	return face.rx
}

func (face *l3faceImpl) GetTx() chan<- L3Packet {
	return face.tx
}

func (face *l3faceImpl) rxLoop() {
	for wire := range face.tr.GetRx() {
		var packet Packet
		e := tlv.Decode(wire, &packet)
		if e != nil {
			continue
		}
		face.rx <- &packet
	}
	close(face.rx)
}

func (face *l3faceImpl) txLoop() {
	transportTx := face.tr.GetTx()
	for l3packet := range face.tx {
		wire, e := tlv.Encode(l3packet.ToPacket())
		if e != nil {
			continue
		}
		transportTx <- wire
	}
	close(transportTx)
}
