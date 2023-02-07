package socketface

import (
	"errors"

	"github.com/gogf/greuse"
	"github.com/usnistgov/ndn-dpdk/iface"
)

const (
	schemeUnix = "unix"
	schemeUDP  = "udp"
	schemeTCP  = "tcp"
)

// Locator describes network and addresses of a socket.
type Locator struct {
	*Config
	Network string `json:"scheme"`
	Local   string `json:"local,omitempty"`
	Remote  string `json:"remote"`
}

// Scheme returns the protocol.
func (loc Locator) Scheme() string {
	return loc.Network
}

// WithSchemeField implements iface.locatorWithSchemeField interface.
func (Locator) WithSchemeField() {}

// Validate checks the addresses.
func (loc Locator) Validate() error {
	_, eR := greuse.ResolveAddr(loc.Network, loc.Remote)
	var eL error
	if loc.Local != "" && !(loc.Network == schemeUnix && loc.Local == "@") {
		_, eL = greuse.ResolveAddr(loc.Network, loc.Local)
	}
	return errors.Join(eR, eL)
}

// CreateFace creates a face from this Locator.
func (loc Locator) CreateFace() (iface.Face, error) {
	return New(loc)
}

func init() {
	iface.RegisterLocatorScheme[Locator](schemeUnix, schemeUDP, schemeTCP)
}
