package ethface

import (
	"errors"

	"github.com/usnistgov/ndn-dpdk/core/macaddr"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/iface/ethport"
	"github.com/usnistgov/ndn-dpdk/ndn/packettransport"
)

// PassthruLocator describes a pass-through face.
type PassthruLocator struct {
	ethport.FaceConfig

	// Local is the local MAC address.
	// This must be a 48-bit unicast address.
	Local macaddr.Flag `json:"local,omitempty"`
}

// Scheme returns "passthru".
func (PassthruLocator) Scheme() string {
	return ethport.SchemePassthru
}

// Validate checks Locator fields.
func (loc PassthruLocator) Validate() error {
	if !loc.Local.Empty() && !macaddr.IsUnicast(loc.Local.HardwareAddr) {
		return packettransport.ErrUnicastMacAddr
	}
	if loc.Local.Empty() && loc.Port == "" && loc.EthDev == nil {
		return errors.New("either local or port must be specified")
	}
	return nil
}

// EthLocatorC implements ethport.Locator interface.
func (loc PassthruLocator) EthLocatorC() (c ethport.LocatorC) {
	c.Remote.Bytes = [6]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF} // C.EthLocator_Classify
	return
}

// CreateFace creates a pass-through face.
func (loc PassthruLocator) CreateFace() (face iface.Face, e error) {
	port, e := loc.FaceConfig.FindPort(loc.Local.HardwareAddr)
	if e != nil {
		return nil, e
	}

	if loc.Local.Empty() {
		loc.Local.HardwareAddr = port.EthDev().HardwareAddr()
	}
	loc.FaceConfig.HideFaceConfigFromJSON()
	return ethport.NewFace(port, loc)
}

func init() {
	iface.RegisterLocatorScheme[PassthruLocator](ethport.SchemePassthru)
}
