package socketface

import (
	"fmt"
	"net"

	"github.com/usnistgov/ndn-dpdk/iface"
)

// Locator.Network values.
const (
	NetworkUnix = "unix"
	NetworkUDP  = "udp"
	NetworkTCP  = "tcp"
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
	switch loc.Network {
	case NetworkUnix:
		if _, e := net.ResolveUnixAddr(loc.Network, loc.Remote); e != nil {
			return fmt.Errorf("remote %w", e)
		}
		if loc.Local != "" && loc.Local != "@" {
			if _, e := net.ResolveUnixAddr(loc.Network, loc.Local); e != nil {
				return fmt.Errorf("local %w", e)
			}
		}
		return nil
	case NetworkUDP:
		if _, e := net.ResolveUDPAddr(loc.Network, loc.Remote); e != nil {
			return fmt.Errorf("remote %w", e)
		}
		if loc.Local != "" {
			if _, e := net.ResolveUDPAddr(loc.Network, loc.Local); e != nil {
				return fmt.Errorf("local %w", e)
			}
		}
		return nil
	case NetworkTCP:
		if _, e := net.ResolveTCPAddr(loc.Network, loc.Remote); e != nil {
			return fmt.Errorf("remote %w", e)
		}
		if loc.Local != "" {
			if _, e := net.ResolveTCPAddr(loc.Network, loc.Local); e != nil {
				return fmt.Errorf("local %w", e)
			}
		}
		return nil
	}
	return fmt.Errorf("unknown scheme %s", loc.Network)
}

// CreateFace creates a face from this Locator.
func (loc Locator) CreateFace() (iface.Face, error) {
	return New(loc)
}

func init() {
	iface.RegisterLocatorType(Locator{}, NetworkUnix, NetworkUDP, NetworkTCP)
}
