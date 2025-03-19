package ethface

import (
	"errors"

	"github.com/usnistgov/ndn-dpdk/core/macaddr"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/iface/ethport"
)

// PassthruLocator describes a pass-through face.
type PassthruLocator struct {
	ethport.FaceConfig

	// Local is the local MAC address.
	// This must be a 48-bit unicast address.
	Local macaddr.Flag `json:"local,omitempty"`

	// Gtpip enables and configures GTP-IP handler.
	Gtpip *ethport.GtpipConfig `json:"gtpip,omitempty"`
}

// Scheme returns "passthru".
func (PassthruLocator) Scheme() string {
	return ethport.SchemePassthru
}

// GtpipConfig returns GtpipConfig if present.
//
// Implements passthruStart.withGtpipConfig interface.
func (loc PassthruLocator) GtpipConfig() *ethport.GtpipConfig {
	return loc.Gtpip
}

// Validate checks Locator fields.
func (loc PassthruLocator) Validate() error {
	if !loc.Local.Empty() && !macaddr.IsUnicast(loc.Local.HardwareAddr) {
		return macaddr.ErrUnicast
	}
	if loc.Local.Empty() && loc.Port == "" && loc.EthDev == nil {
		return errors.New("either local or port must be specified")
	}
	return nil
}

// EthLocatorC implements ethport.Locator interface.
func (loc PassthruLocator) EthLocatorC() (c ethport.LocatorC) {
	// C.EthLocator_Classify interprets the broadcast address as pass-through face
	c.Remote.Bytes = [6]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
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
