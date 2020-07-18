package packettransport

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"

	"github.com/usnistgov/ndn-dpdk/core/macaddr"
)

const (
	// MinVLAN is the minimum VLAN number.
	MinVLAN = 0x001

	// MaxVLAN is the maximum VLAN number.
	MaxVLAN = 0xFFF

	// EthernetTypeNDN is the NDN EtherType.
	EthernetTypeNDN = 0x8624
)

var (
	// MulticastAddressNDN is the default NDN multicast address.
	MulticastAddressNDN = net.HardwareAddr{0x01, 0x00, 0x5E, 0x00, 0x17, 0xAA}
)

// Locator identifies local and remote endpoints.
type Locator struct {
	// Local is the local MAC address.
	// This must be a 48-bit unicast address.
	Local net.HardwareAddr

	// Remote is the remote MAC address.
	// This must be a 48-bit unicast or multicast address.
	Remote net.HardwareAddr

	// VLAN is the VLAN number.
	// This must be between MinVLAN and MaxVLAN.
	// Zero indicates there's no VLAN header.
	VLAN int
}

// Validate checks Locator fields.
func (loc Locator) Validate() error {
	if !macaddr.IsUnicast(loc.Local) {
		return errors.New("invalid Local")
	}
	if !macaddr.IsUnicast(loc.Remote) && !macaddr.IsMulticast(loc.Remote) {
		return errors.New("invalid Remote")
	}
	if loc.VLAN != 0 && (loc.VLAN < MinVLAN || loc.VLAN > MaxVLAN) {
		return errors.New("invalid VLAN")
	}
	return nil
}

// MarshalJSON implements json.Marshaler interface.
func (loc Locator) MarshalJSON() ([]byte, error) {
	if e := loc.Validate(); e != nil {
		return nil, e
	}
	return json.Marshal(locatorJSON{
		Local:  loc.Local.String(),
		Remote: loc.Remote.String(),
		VLAN:   loc.VLAN,
	})
}

// UnmarshalJSON implements json.Unmarshaler interface.
func (loc *Locator) UnmarshalJSON(data []byte) (e error) {
	var j locatorJSON
	if e = json.Unmarshal(data, &j); e != nil {
		return e
	}
	if loc.Local, e = net.ParseMAC(j.Local); e != nil {
		return fmt.Errorf("Local %w", e)
	}
	if loc.Remote, e = net.ParseMAC(j.Remote); e != nil {
		return fmt.Errorf("Remote %w", e)
	}
	loc.VLAN = j.VLAN
	return loc.Validate()
}

type locatorJSON struct {
	Local  string `json:"local"`
	Remote string `json:"remote"`
	VLAN   int    `json:"vlan,omitempty"`
}
