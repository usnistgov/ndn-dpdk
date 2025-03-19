package ethface

import (
	"errors"
	"math"
	"net/netip"

	"github.com/usnistgov/ndn-dpdk/core/macaddr"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/iface/ethport"
)

// Error conditions.
var (
	ErrIP        = errors.New("invalid IP address")
	ErrIPFamily  = errors.New("different address family in LocalIP and RemoteIP")
	ErrUnicastIP = errors.New("invalid unicast IP address")
	ErrUDPPort   = errors.New("invalid UDP port")
)

// IPLocator describes an IP-based face.
type IPLocator struct {
	// EtherLocator contains MAC addresses and EthDev specification.
	// loc.Remote must be a unicast address.
	EtherLocator

	// LocalIP is the local IP address.
	// It may be either IPv4 or IPv6.
	LocalIP netip.Addr `json:"localIP"`

	// RemoteIP is the remote IP address.
	// It may be either IPv4 or IPv6.
	RemoteIP netip.Addr `json:"remoteIP"`
}

// Validate checks Locator fields.
func (loc IPLocator) Validate() error {
	if e := loc.EtherLocator.Validate(); e != nil {
		return e
	}

	local, remote := loc.LocalIP.Unmap(), loc.RemoteIP.Unmap()
	switch {
	case !macaddr.IsUnicast(loc.Remote.HardwareAddr):
		return macaddr.ErrUnicast
	case local.BitLen() == 0, remote.BitLen() == 0:
		return ErrIP
	case local.BitLen() != remote.BitLen():
		return ErrIPFamily
	case local.IsMulticast(), remote.IsMulticast():
		return ErrUnicastIP
	}

	return nil
}

func (loc IPLocator) ipLocatorC() (locC ethport.LocatorC) {
	locC = loc.EtherLocator.EthLocatorC()
	locC.LocalIP.A = loc.LocalIP.As16()
	locC.RemoteIP.A = loc.RemoteIP.As16()
	return
}

const schemeUDP = "udpe"

// UDPLocator describes a UDP face.
type UDPLocator struct {
	IPLocator

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
	if e := loc.IPLocator.Validate(); e != nil {
		return e
	}

	switch {
	case loc.LocalUDP <= 0 || loc.LocalUDP > math.MaxUint16,
		loc.RemoteUDP <= 0 || loc.RemoteUDP > math.MaxUint16:
		return ErrUDPPort
	}

	return nil
}

// EthLocatorC implements ethport.Locator interface.
func (loc UDPLocator) EthLocatorC() (locC ethport.LocatorC) {
	locC = loc.IPLocator.ipLocatorC()
	locC.LocalUDP = uint16(loc.LocalUDP)
	locC.RemoteUDP = uint16(loc.RemoteUDP)
	return
}

// CreateFace creates a UDP face.
func (loc UDPLocator) CreateFace() (face iface.Face, e error) {
	port, e := loc.FaceConfig.FindPort(loc.Local.HardwareAddr)
	if e != nil {
		return nil, e
	}

	loc.FaceConfig.HideFaceConfigFromJSON()
	return ethport.NewFace(port, loc)
}

func init() {
	iface.RegisterLocatorScheme[UDPLocator](schemeUDP)
}
