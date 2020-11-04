package ethface

import (
	"encoding/binary"
	"errors"

	"github.com/koneu/natend"
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

	// PortConfig specifies additional configuration for Port activation.
	// This is only used when creating the first face on an EthDev.
	PortConfig *PortConfig `json:"portConfig,omitempty"`
}

// Scheme returns "ether".
func (loc EtherLocator) Scheme() string {
	return schemeEther
}

func (loc EtherLocator) conflictsWith(other ethLocator) bool {
	r, ok := other.(EtherLocator)
	return !ok ||
		(macaddr.IsMulticast(loc.Remote.HardwareAddr) && macaddr.IsMulticast(r.Remote.HardwareAddr)) ||
		macaddr.Equal(loc.Remote.HardwareAddr, r.Remote.HardwareAddr)
}

func (loc EtherLocator) cLoc() (c cLocator) {
	copy(c.Local.Bytes[:], []uint8(loc.Local.HardwareAddr))
	copy(c.Remote.Bytes[:], []uint8(loc.Remote.HardwareAddr))
	var vlan [2]byte
	binary.BigEndian.PutUint16(vlan[:], uint16(loc.VLAN))
	c.Vlan = natend.NativeEndian.Uint16(vlan[:])
	return
}

// CreateFace creates an Ethernet face.
func (loc EtherLocator) CreateFace() (face iface.Face, e error) {
	dev := loc.findEthDev()
	if !dev.Valid() {
		return nil, ErrNoPort
	}
	port := FindPort(dev)

	if port == nil {
		var cfg PortConfig
		if loc.PortConfig != nil {
			cfg = *loc.PortConfig
		}
		if port, e = NewPort(dev, cfg); e != nil {
			return nil, e
		}
	}

	loc.Port = port.dev.Name()
	loc.PortConfig = nil
	return New(port, loc)
}

func (loc EtherLocator) findEthDev() ethdev.EthDev {
	for _, dev := range ethdev.List() {
		if loc.Port == "" {
			if macaddr.Equal(dev.MacAddr(), loc.Local.HardwareAddr) {
				return dev
			}
		} else if dev.Name() == loc.Port {
			return dev
		}
	}
	return ethdev.EthDev{}
}

func init() {
	iface.RegisterLocatorType(EtherLocator{}, schemeEther)
}
