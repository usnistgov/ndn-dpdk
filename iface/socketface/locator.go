package socketface

import (
	"fmt"
	"net"

	"github.com/usnistgov/ndn-dpdk/iface"
)

// Locator describes local and remote address of a socket.
type Locator struct {
	iface.LocatorBase
	Local  string
	Remote string
}

// Validate checks the addresses.
func (loc Locator) Validate() error {
	switch loc.Scheme {
	case "unix":
		if _, e := net.ResolveUnixAddr(loc.Scheme, loc.Remote); e != nil {
			return fmt.Errorf("remote %w", e)
		}
		if loc.Local != "" && loc.Local != "@" {
			if _, e := net.ResolveUnixAddr(loc.Scheme, loc.Local); e != nil {
				return fmt.Errorf("remote %w", e)
			}
		}
		return nil
	case "udp":
		if _, e := net.ResolveUDPAddr(loc.Scheme, loc.Remote); e != nil {
			return fmt.Errorf("remote %w", e)
		}
		if loc.Local != "" {
			if _, e := net.ResolveUDPAddr(loc.Scheme, loc.Local); e != nil {
				return fmt.Errorf("local %w", e)
			}
		}
		return nil
	case "tcp":
		if _, e := net.ResolveTCPAddr(loc.Scheme, loc.Remote); e != nil {
			return fmt.Errorf("remote %w", e)
		}
		if loc.Local != "" {
			if _, e := net.ResolveTCPAddr(loc.Scheme, loc.Local); e != nil {
				return fmt.Errorf("local %w", e)
			}
		}
		return nil
	}
	return fmt.Errorf("unknown scheme %s", loc.Scheme)
}

func init() {
	iface.RegisterLocatorType(Locator{}, "udp")
	iface.RegisterLocatorType(Locator{}, "tcp")
	iface.RegisterLocatorType(Locator{}, "unix")
}
