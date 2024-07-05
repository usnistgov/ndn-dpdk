package ethface

import (
	"errors"

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
	Local macaddr.Flag `json:"local,omitempty"`
}

// Scheme returns "fallback".
func (FallbackLocator) Scheme() string {
	return ethport.SchemeFallback
}

// Validate checks Locator fields.
func (loc FallbackLocator) Validate() error {
	if !loc.Local.Empty() && !macaddr.IsUnicast(loc.Local.HardwareAddr) {
		return packettransport.ErrUnicastMacAddr
	}
	if loc.Local.Empty() && loc.Port == "" && loc.EthDev == nil {
		return errors.New("either local or port must be specifed")
	}
	return nil
}

// EthLocatorC implements ethport.Locator interface.
func (loc FallbackLocator) EthLocatorC() (c ethport.LocatorC) {
	c.Remote.Bytes = [6]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF} // C.EthLocator_Classify
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
	iface.RegisterLocatorScheme[FallbackLocator](ethport.SchemeFallback)
}
