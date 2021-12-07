package ethface

import (
	"errors"

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

	// Port is the EthDev name.
	//
	// During face creation, if this field is empty:
	// * Face is created on an Ethernet adapter whose physical MAC address equals loc.Local.
	// * Local MAC address of the face is set to loc.Local, i.e. same as the physical MAC address.
	//
	// During face creation, if this field is non-empty:
	// * Face is created on an Ethernet adapter whose DPDK EthDev name equals loc.Port.
	// * Local MAC address of the face is set to loc.Local, which could differ from the physical MAC address.
	//
	// When retrieving face information, this reflects the DPDK EthDev name.
	Port string `json:"port,omitempty"`
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
	var dev ethdev.EthDev
	if loc.Port == "" {
		dev = ethdev.FromHardwareAddr(loc.Local.HardwareAddr)
	} else {
		dev = ethdev.FromName(loc.Port)
	}
	if dev != nil {
		port = portByEthDev[dev]
	}

	if port == nil {
		return nil, errors.New("Port does not exist; Port must be created before creating face")
	}

	loc.Port = dev.Name()
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
