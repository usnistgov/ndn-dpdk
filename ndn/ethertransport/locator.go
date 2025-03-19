// Package ethertransport implements a transport over AF_PACKET socket.
package ethertransport

import (
	"net"

	"github.com/usnistgov/ndn-dpdk/core/macaddr"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
)

// EthernetTypeNDN is the NDN EtherType.
const EthernetTypeNDN = an.EtherTypeNDN

// MulticastAddressNDN is the default NDN multicast address.
var MulticastAddressNDN = net.HardwareAddr{
	an.EtherMulticastNDN >> 40 & 0xFF,
	an.EtherMulticastNDN >> 32 & 0xFF,
	an.EtherMulticastNDN >> 24 & 0xFF,
	an.EtherMulticastNDN >> 16 & 0xFF,
	an.EtherMulticastNDN >> 8 & 0xFF,
	an.EtherMulticastNDN >> 0 & 0xFF,
}

// Locator identifies local and remote endpoints.
type Locator struct {
	// Local is the local MAC address.
	// This must be a 48-bit unicast address.
	Local macaddr.Flag `json:"local"`

	// Remote is the remote MAC address.
	// This must be a 48-bit unicast or multicast address.
	Remote macaddr.Flag `json:"remote"`

	// VLAN is the VLAN identifier.
	// This must be between 1 and 4094.
	// Zero indicates the absence of a VLAN header.
	VLAN int `json:"vlan,omitempty"`
}

// Validate checks Locator fields.
func (loc Locator) Validate() error {
	if !macaddr.IsUnicast(loc.Local.HardwareAddr) {
		return macaddr.ErrUnicast
	}
	if !macaddr.IsUnicast(loc.Remote.HardwareAddr) && !macaddr.IsMulticast(loc.Remote.HardwareAddr) {
		return macaddr.ErrAddr
	}
	if loc.VLAN != 0 && !macaddr.IsVLAN(loc.VLAN) {
		return macaddr.ErrVLAN
	}
	return nil
}
