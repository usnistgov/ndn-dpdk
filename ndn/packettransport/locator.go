package packettransport

import (
	"errors"
	"net"

	"github.com/usnistgov/ndn-dpdk/core/macaddr"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
)

const (
	// MinVLAN is the minimum VLAN number.
	MinVLAN = 0x001

	// MaxVLAN is the maximum VLAN number.
	MaxVLAN = 0xFFE

	// EthernetTypeNDN is the NDN EtherType.
	EthernetTypeNDN = an.EtherTypeNDN
)

// MulticastAddressNDN is the default NDN multicast address.
var MulticastAddressNDN = net.HardwareAddr{
	an.EtherMulticastNDN >> 40 & 0xFF,
	an.EtherMulticastNDN >> 32 & 0xFF,
	an.EtherMulticastNDN >> 24 & 0xFF,
	an.EtherMulticastNDN >> 16 & 0xFF,
	an.EtherMulticastNDN >> 8 & 0xFF,
	an.EtherMulticastNDN >> 0 & 0xFF,
}

// Error conditions.
var (
	ErrMacAddr        = errors.New("invalid MAC address")
	ErrUnicastMacAddr = errors.New("invalid unicast MAC address")
	ErrVLAN           = errors.New("invalid VLAN")
)

// Locator identifies local and remote endpoints.
type Locator struct {
	// Local is the local MAC address.
	// This must be a 48-bit unicast address.
	Local macaddr.Flag `json:"local"`

	// Remote is the remote MAC address.
	// This must be a 48-bit unicast or multicast address.
	Remote macaddr.Flag `json:"remote"`

	// VLAN is the VLAN number.
	// This must be between MinVLAN and MaxVLAN.
	// Zero indicates there's no VLAN header.
	VLAN int `json:"vlan,omitempty"`
}

// Validate checks Locator fields.
func (loc Locator) Validate() error {
	if !macaddr.IsUnicast(loc.Local.HardwareAddr) {
		return ErrUnicastMacAddr
	}
	if !macaddr.IsUnicast(loc.Remote.HardwareAddr) && !macaddr.IsMulticast(loc.Remote.HardwareAddr) {
		return ErrMacAddr
	}
	if loc.VLAN != 0 && (loc.VLAN < MinVLAN || loc.VLAN > MaxVLAN) {
		return ErrVLAN
	}
	return nil
}
