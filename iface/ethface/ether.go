// Package ethface implements Ethernet-based faces.
package ethface

import (
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/iface/ethport"
	"github.com/usnistgov/ndn-dpdk/ndn/ethertransport"
)

const schemeEther = "ether"

// EtherLocator describes an Ethernet face.
type EtherLocator struct {
	ethport.FaceConfig

	// ethertransport.Locator contains MAC addresses.
	ethertransport.Locator
}

// Scheme returns "ether".
func (EtherLocator) Scheme() string {
	return schemeEther
}

// EthLocatorC implements ethport.Locator interface.
func (loc EtherLocator) EthLocatorC() (c ethport.LocatorC) {
	copy(c.Local.Bytes[:], []uint8(loc.Local.HardwareAddr))
	copy(c.Remote.Bytes[:], []uint8(loc.Remote.HardwareAddr))
	c.Vlan = uint16(loc.VLAN)
	return
}

// CreateFace creates an Ethernet face.
func (loc EtherLocator) CreateFace() (face iface.Face, e error) {
	port, e := loc.FaceConfig.FindPort(loc.Local.HardwareAddr)
	if e != nil {
		return nil, e
	}

	loc.FaceConfig.HideFaceConfigFromJSON()
	return ethport.NewFace(port, loc)
}

func init() {
	iface.RegisterLocatorScheme[EtherLocator](schemeEther)
}
