package ethface

import (
	"errors"

	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn/packettransport"
)

// Error conditions.
var (
	ErrNoPort = errors.New("EthDev not found")
)

const schemeEther = "ether"

// EtherLocator describes an Ethernet face.
type EtherLocator struct {
	FaceConfig

	// packettransport.Locator contains MAC addresses.
	packettransport.Locator
}

// Scheme returns "ether".
func (EtherLocator) Scheme() string {
	return schemeEther
}

func (loc EtherLocator) cLoc() (c cLocator) {
	copy(c.Local.Bytes[:], []uint8(loc.Local.HardwareAddr))
	copy(c.Remote.Bytes[:], []uint8(loc.Remote.HardwareAddr))
	c.Vlan = uint16(loc.VLAN)
	return
}

func (loc *EtherLocator) findPort() (port *Port, e error) {
	dev := loc.EthDev
	switch {
	case dev != nil:
	case loc.Port != "":
		gqlserver.RetrieveNodeOfType(ethdev.GqlEthDevNodeType, loc.Port, &dev)
	default:
		dev = ethdev.FromHardwareAddr(loc.Local.HardwareAddr)
	}

	if dev != nil {
		port = portByEthDev[dev]
	}
	if port == nil {
		return nil, errors.New("Port does not exist; Port must be created before creating face")
	}

	loc.FaceConfig = loc.FaceConfig.hideFaceConfigFromJSON()
	return port, nil
}

// CreateFace creates an Ethernet face.
func (loc EtherLocator) CreateFace() (face iface.Face, e error) {
	port, e := loc.findPort()
	if e != nil {
		return nil, e
	}
	return New(port, loc)
}

func init() {
	iface.RegisterLocatorType(EtherLocator{}, schemeEther)
}
