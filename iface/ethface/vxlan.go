package ethface

import (
	"errors"

	"github.com/usnistgov/ndn-dpdk/core/macaddr"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/iface/ethport"
	"github.com/usnistgov/ndn-dpdk/ndn/packettransport"
)

const (
	// MinVXLAN is the minimum VXLAN Network Identifier.
	MinVXLAN = 0

	// MaxVXLAN is the maximum VXLAN Network Identifier.
	MaxVXLAN = 1<<24 - 1
)

// Error conditions.
var (
	ErrVXLAN = errors.New("invalid VXLAN Network Identifier")
)

const schemeVxlan = "vxlan"

// VxlanLocator describes an Ethernet VXLAN face.
type VxlanLocator struct {
	IPLocator

	// VXLAN is the VXLAN virtual network identifier.
	// This must be between MinVXLAN and MaxVXLAN.
	VXLAN int `json:"vxlan"`

	// InnerLocal is the inner local MAC address.
	// This must be a 48-bit unicast address.
	InnerLocal macaddr.Flag `json:"innerLocal"`

	// InnerRemote is the inner remote MAC address.
	// This must be a 48-bit unicast address.
	InnerRemote macaddr.Flag `json:"innerRemote"`
}

// Scheme returns "vxlan".
func (VxlanLocator) Scheme() string {
	return schemeVxlan
}

// Validate checks Locator fields.
func (loc VxlanLocator) Validate() error {
	if e := loc.IPLocator.Validate(); e != nil {
		return e
	}

	switch {
	case loc.VXLAN < MinVXLAN, loc.VXLAN > MaxVXLAN:
		return ErrVXLAN
	case !macaddr.IsUnicast(loc.InnerLocal.HardwareAddr), !macaddr.IsUnicast(loc.InnerRemote.HardwareAddr):
		return packettransport.ErrUnicastMacAddr
	}

	return nil
}

// EthLocatorC implements ethport.Locator interface.
func (loc VxlanLocator) EthLocatorC() (locC ethport.LocatorC) {
	locC = loc.IPLocator.ipLocatorC()
	locC.LocalUDP = ethport.UDPPortVXLAN
	locC.RemoteUDP = ethport.UDPPortVXLAN
	locC.Vxlan = uint32(loc.VXLAN)
	copy(locC.InnerLocal.Bytes[:], loc.InnerLocal.HardwareAddr)
	copy(locC.InnerRemote.Bytes[:], loc.InnerRemote.HardwareAddr)
	return
}

// CreateFace creates a VXLAN face.
func (loc VxlanLocator) CreateFace() (face iface.Face, e error) {
	port, e := loc.FaceConfig.FindPort(loc.Local.HardwareAddr)
	if e != nil {
		return nil, e
	}

	loc.FaceConfig.HideFaceConfigFromJSON()
	return ethport.NewFace(port, loc)
}

func init() {
	iface.RegisterLocatorScheme[VxlanLocator](schemeVxlan)
}
