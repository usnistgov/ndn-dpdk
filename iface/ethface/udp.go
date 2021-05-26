package ethface

import (
	"errors"
	"math"

	"github.com/usnistgov/ndn-dpdk/core/macaddr"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn/packettransport"
	"inet.af/netaddr"
)

// Error conditions.
var (
	ErrIP        = errors.New("invalid IP address")
	ErrIPFamily  = errors.New("different address family in LocalIP and RemoteIP")
	ErrUnicastIP = errors.New("invalid unicast IP address")
	ErrUDPPort   = errors.New("invalid UDP port")
)

const schemeUDP = "udpe"

// UDPLocator describes a UDP face.
type UDPLocator struct {
	// EtherLocator contains MAC addresses and EthDev specification.
	// loc.Remote must be a unicast address.
	EtherLocator

	// LocalIP is the local IP address.
	// It may be either IPv4 or IPv6.
	LocalIP netaddr.IP `json:"localIP"`

	// RemoteIP is the remote IP address.
	// It may be either IPv4 or IPv6.
	RemoteIP netaddr.IP `json:"remoteIP"`

	// LocalUDP is the local UDP port number.
	LocalUDP int `json:"localUDP"`

	// RemoteUDP is the remote UDP port number.
	RemoteUDP int `json:"remoteUDP"`
}

// Scheme returns "udpe".
func (UDPLocator) Scheme() string {
	return schemeUDP
}

// Validate checks Locator fields.
func (loc UDPLocator) Validate() error {
	if e := loc.EtherLocator.Validate(); e != nil {
		return e
	}

	local, remote := loc.LocalIP.Unmap(), loc.RemoteIP.Unmap()
	switch {
	case !macaddr.IsUnicast(loc.Remote.HardwareAddr):
		return packettransport.ErrUnicastMacAddr
	case local.IsZero(), remote.IsZero():
		return ErrIP
	case local.BitLen() != remote.BitLen():
		return ErrIPFamily
	case local.IsMulticast(), remote.IsMulticast():
		return ErrUnicastIP
	case loc.LocalUDP <= 0 || loc.LocalUDP > math.MaxUint16,
		loc.RemoteUDP <= 0 || loc.RemoteUDP > math.MaxUint16:
		return ErrUDPPort
	}

	return nil
}

func (loc UDPLocator) cLoc() (c cLocator) {
	c = loc.EtherLocator.cLoc()
	c.LocalIP = loc.LocalIP.As16()
	c.RemoteIP = loc.RemoteIP.As16()
	c.LocalUDP = uint16(loc.LocalUDP)
	c.RemoteUDP = uint16(loc.RemoteUDP)
	return
}

// CreateFace creates a UDP face.
func (loc UDPLocator) CreateFace() (face iface.Face, e error) {
	port, e := loc.makePort()
	if e != nil {
		return nil, e
	}
	return New(port, loc)
}

func init() {
	iface.RegisterLocatorType(UDPLocator{}, schemeUDP)
}
