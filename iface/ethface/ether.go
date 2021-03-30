package ethface

import (
	"errors"

	"github.com/usnistgov/ndn-dpdk/core/macaddr"
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
	// During face creation:
	//  * If this is empty:
	//    * Face is created on an EthDev whose physical MAC address matches loc.Local.
	//    * Local MAC address of the face is set to loc.Local, i.e. same as the physical MAC address.
	//  * If this is non-empty:
	//    * Face is created on an EthDev whose name matches loc.Port.
	//    * Local MAC address of the face is set to loc.Local, which could differ from the physical MAC address.
	//  * In either case, if no matching EthDev is found, face creation fails.
	//
	// When retrieving face information, this reflects the EthDev name.
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

// CreateFace creates an Ethernet face.
func (loc EtherLocator) CreateFace() (face iface.Face, e error) {
	port, e := loc.makePort()
	if e != nil {
		return nil, e
	}
	return New(port, loc)
}

func (loc *EtherLocator) makePort() (port *Port, e error) {
	dev := loc.findEthDev()
	if dev == nil {
		return nil, ErrNoPort
	}
	port = FindPort(dev)

	if port == nil {
		var cfg PortConfig
		if loc.PortConfig != nil {
			cfg = *loc.PortConfig
		}
		if port, e = NewPort(dev, cfg); e != nil {
			return nil, e
		}
	}

	loc.Port = dev.Name()
	loc.FaceConfig = loc.FaceConfig.hideFaceConfigFromJSON()
	return port, nil
}

func (loc EtherLocator) findEthDev() ethdev.EthDev {
	for _, dev := range ethdev.List() {
		if loc.Port == "" {
			if macaddr.Equal(dev.MACAddr(), loc.Local.HardwareAddr) {
				return dev
			}
		} else if dev.Name() == loc.Port {
			return dev
		}
	}
	return nil
}

func init() {
	iface.RegisterLocatorType(EtherLocator{}, schemeEther)
}
