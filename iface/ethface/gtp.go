package ethface

import (
	"errors"
	"math"
	"net/netip"

	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/iface/ethport"
)

// Error conditions.
var (
	ErrTEID = errors.New("invalid Tunnel Endpoint Identifier")
	ErrQFI  = errors.New("invalid QoS Flow Identifier")
)

const schemeGtp = "gtp"
const maxQFI = 0b111111

// GtpLocator describes a GTP-U face.
type GtpLocator struct {
	IPLocator

	// UlTEID is the uplink/incoming tunnel endpoint identifier.
	// This must fit in 32 bits.
	UlTEID int `json:"ulTEID"`

	// UlQFI is the uplink/incoming QoS flow identifier.
	// This must fit in 6 bits.
	UlQFI int `json:"ulQFI"`

	// DlTEID is the downlink/outgoing tunnel endpoint identifier.
	// This must fit in 32 bits.
	DlTEID int `json:"dlTEID"`

	// DlQFI is the downlink/outgoing QoS flow identifier.
	// This must fit in 6 bits.
	DlQFI int `json:"dlQFI"`

	// InnerLocalIP is the inner local IPv4 address.
	InnerLocalIP netip.Addr `json:"innerLocalIP"`

	// InnerRemoteIP is the inner remote IPv4 address.
	InnerRemoteIP netip.Addr `json:"innerRemoteIP"`
}

// Scheme returns "gtp".
func (GtpLocator) Scheme() string {
	return schemeGtp
}

// Validate checks Locator fields.
func (loc GtpLocator) Validate() error {
	if e := loc.IPLocator.Validate(); e != nil {
		return e
	}

	local, remote := loc.InnerLocalIP.Unmap(), loc.InnerRemoteIP.Unmap()
	switch {
	case loc.UlTEID < 0, loc.UlTEID > math.MaxUint32, loc.DlTEID < 0, loc.DlTEID > math.MaxUint32:
		return ErrTEID
	case loc.UlQFI < 0, loc.UlQFI > maxQFI, loc.DlQFI < 0, loc.DlQFI > maxQFI:
		return ErrQFI
	case !local.Is4(), local.IsMulticast(), !remote.Is4(), remote.IsMulticast():
		return ErrUnicastIP
	}

	return nil
}

// EthLocatorC implements ethport.Locator interface.
func (loc GtpLocator) EthLocatorC() (locC ethport.LocatorC) {
	locC = loc.IPLocator.ipLocatorC()
	locC.LocalUDP = ethport.UDPPortGTP
	locC.RemoteUDP = ethport.UDPPortGTP
	locC.IsGtp = true
	locC.UlTEID = uint32(loc.UlTEID)
	locC.UlQFI = uint8(loc.UlQFI)
	locC.DlTEID = uint32(loc.DlTEID)
	locC.DlQFI = uint8(loc.DlQFI)
	locC.InnerLocalIP.A = loc.InnerLocalIP.As16()
	locC.InnerRemoteIP.A = loc.InnerRemoteIP.As16()
	return
}

// CreateFace creates a GTP-U face.
func (loc GtpLocator) CreateFace() (face iface.Face, e error) {
	port, e := loc.FaceConfig.FindPort(loc.Local.HardwareAddr)
	if e != nil {
		return nil, e
	}

	loc.FaceConfig.HideFaceConfigFromJSON()
	return ethport.NewFace(port, loc)
}

func init() {
	iface.RegisterLocatorScheme[GtpLocator](schemeGtp)
}
