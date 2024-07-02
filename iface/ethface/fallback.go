package ethface

import (
	"github.com/usnistgov/ndn-dpdk/core/macaddr"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/iface/ethport"
	"github.com/usnistgov/ndn-dpdk/ndn/packettransport"
)

// FallbackLocator describes a fallback face.
type FallbackLocator struct {
	ethport.FaceConfig

	// Local is the local MAC address.
	// This must be a 48-bit unicast address.
	Local macaddr.Flag `json:"local"`
}

// Scheme returns "fallback".
func (FallbackLocator) Scheme() string {

	return ethport.FallbackScheme
}

// Validate checks Locator fields.
func (loc FallbackLocator) Validate() error {
	if !macaddr.IsUnicast(loc.Local.HardwareAddr) {
		return packettransport.ErrUnicastMacAddr
	}
	return nil
}

// EthLocatorC implements ethport.Locator interface.
func (loc FallbackLocator) EthLocatorC() (c ethport.LocatorC) {
	c.Remote.Bytes = ethport.FallbackRemoteSentinel
	return
}

// CreateFace creates a fallback face.
func (loc FallbackLocator) CreateFace() (face iface.Face, e error) {
	port, e := loc.FaceConfig.FindPort(loc.Local.HardwareAddr)
	if e != nil {
		return nil, e
	}

	loc.FaceConfig.HideFaceConfigFromJSON()
	return ethport.NewFace(port, loc)
}

func init() {
	iface.RegisterLocatorScheme[FallbackLocator](ethport.FallbackScheme)
}
