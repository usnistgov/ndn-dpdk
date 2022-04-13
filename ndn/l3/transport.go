package l3

import "io"

// Transport represents a communicate channel to send and receive TLV packets.
type Transport interface {
	io.ReadWriteCloser

	// MTU returns maximum size of outgoing packets.
	MTU() int

	// State returns current state.
	State() TransportState

	// OnStateChange registers a callback to be invoked when State() changes.
	// Returns a function to cancel the callback registration.
	OnStateChange(cb func(st TransportState)) (cancel func())
}

// TransportState indicates up/down state of a transport.
type TransportState int

const (
	// TransportUp indicates the transport is operational.
	TransportUp TransportState = iota

	// TransportDown indicates the transport is nonoperational.
	TransportDown

	// TransportClosed indicates the transport has been closed.
	// It cannot be restarted.
	TransportClosed
)
