package ndn

import (
	"io"
)

// L3Face represents a communicate channel to send and receive NDN packets.
type L3Face interface {
	io.Closer

	// GetRx returns a channel to receive incoming packets.
	// If this function is called multiple times, the same channel should be returned.
	GetRx() <-chan *Packet

	// GetTx returns a channel to send outgoing packets.
	// If this function is called multiple times, the same channel should be returned.
	GetTx() chan<- *Packet
}

// L3FaceAdvertiser is an L3Face that supports advertising prefixes.
type L3FaceAdvertiser interface {
	L3Face

	// Advertise causes the network to send Interests under the specified name prefix toward
	// the current application via the face.
	Advertise(name Name) (withdraw io.Closer, e error)
}
