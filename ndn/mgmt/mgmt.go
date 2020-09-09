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

	// AddRoute adds a route.
	// Adding the same route more than once is not an error but has no effect.
	AddRoute(prefix ndn.Name) error

	// RemoveRoute removes a route.
	// Removing a non-existent route is not an error but has no effect.
	RemoveRoute(prefix ndn.Name) error
}
