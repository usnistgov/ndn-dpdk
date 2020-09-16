// Package mgmt defines interface of forwarder management features.
package mgmt

import (
	"io"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
)

// Client provides access to forwarder management features for the application.
type Client interface {
	io.Closer

	// OpenFace creates a face connected to the current application.
	OpenFace() (Face, error)
}

// Face represents a face.
type Face interface {
	// ID returns face identifier.
	ID() string

	// Face returns an l3.Face.
	Face() l3.Face

	// Close requests the face to be closed.
	Close() error

	// Advertise advertises a prefix announcement.
	// The connected forwarder should start delivering Interests matching this prefix to this face.
	// Advertising the same name more than once is not an error but has no effect.
	Advertise(name ndn.Name) error

	// Withdraw removes a route.
	// The connected forwarder should stop delivering Interests matching this prefix to this face.
	// Withdrawing an unadvertised name is not an error but has no effect.
	Withdraw(name ndn.Name) error
}
