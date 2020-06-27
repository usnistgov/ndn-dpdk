package ndn

import (
	"io"

	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

// Transport represents a communicate channel to send and receive TLV packets.
type Transport interface {
	// Closer allows closing the transport.
	io.Closer

	// GetRx returns a channel to receive incoming TLV elements.
	// It always returns the same channel.
	// This channel should be closed when the transport is closed.
	GetRx() <-chan []byte

	// GetTx returns a channel to send outgoing TLV elements.
	// It always returns the same channel.
	// This channel should remain open when the transport is closed.
	GetTx() chan<- []byte
}

// L3Face represents a communicate channel to send and receive TLV packets.
type L3Face interface {
	// Closer allows closing the face.
	io.Closer

	// GetTransport returns the underlying transport.
	GetTransport() Transport

	// GetRx returns a channel to receive incoming packets.
	// It always returns the same channel.
	// This channel should be closed when the face is closed.
	GetRx() <-chan *Packet

	// GetTx returns a channel to send outgoing packets.
	// It always returns the same channel.
	// This channel should remain open when the transport is closed.
	GetTx() chan<- *Packet
}

// NewL3Face creates an L3Face.
func NewL3Face(tr Transport) (l3face L3Face, e error) {
	var face l3faceImpl
	face.tr = tr
	face.rx = make(chan *Packet)
	face.tx = make(chan *Packet)
	go face.rxLoop()
	go face.txLoop()
	return &face, nil
}

type l3faceImpl struct {
	tr Transport
	rx chan *Packet
	tx chan *Packet
}

func (face *l3faceImpl) Close() error {
	return face.tr.Close()
}

func (face *l3faceImpl) GetTransport() Transport {
	return face.tr
}

func (face *l3faceImpl) GetRx() <-chan *Packet {
	return face.rx
}

func (face *l3faceImpl) GetTx() chan<- *Packet {
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
	for packet := range face.tx {
		wire, e := tlv.Encode(packet)
		if e != nil {
			continue
		}
		transportTx <- wire
	}
}
