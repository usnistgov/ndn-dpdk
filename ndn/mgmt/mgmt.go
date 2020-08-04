// Package mgmt defines interface of forwarder management features.
package mgmt

import (
	"io"

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
	io.Closer

	ID() string

	Face() l3.Face
}
