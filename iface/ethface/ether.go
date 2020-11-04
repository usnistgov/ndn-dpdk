package ethface

import (
	"encoding/binary"
	"errors"
	"net"

	"github.com/koneu/natend"
	"github.com/usnistgov/ndn-dpdk/core/macaddr"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn/packettransport"
)

const schemeEther = "ether"

// EtherLocator describes an Ethernet face.
type EtherLocator struct {
	packettransport.Locator

	// Port is the port name.
	//
	// During face creation, this field is optional.
	// If this is empty, the face creation procedure:
	// (1) Search for an active Port whose local address matches loc.Local.
	// (2) If no such Port was found, search for an inactive EthDev whose physical MAC address matches loc.Local.
	// (3) Face creation fails if no such EthDev was found.
	//
	// If this is non-empty, the face creation procedure:
	// (1) Search for an EthDev whose name matches this value.
	//     Face creation fails if no such EthDev was found.
	// (2) If that EthDev is inactive, a Port is activated with the local address specified in loc.Local.
	//     Otherwise, loc.Local must match the local address of the active Port, or face creation fails.
	Port string `json:"port,omitempty"`

	// PortConfig specifies additional configuration for Port activation.
	// This is only used when creating the first face on an EthDev.
	PortConfig *PortConfig `json:"portConfig,omitempty"`
}

// Scheme returns "ether".
func (loc EtherLocator) Scheme() string {
	return schemeEther
}

func (loc EtherLocator) local() net.HardwareAddr {
	return loc.Local.HardwareAddr
}

func (loc EtherLocator) remote() net.HardwareAddr {
	return loc.Remote.HardwareAddr
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
	dev, port := loc.findPort()
	if !dev.Valid() {
		return nil, errors.New("EthDev not found")
	}

	if port == nil {
		var cfg PortConfig
		if loc.PortConfig != nil {
			cfg = *loc.PortConfig
		}
		if port, e = NewPort(dev, loc.local(), cfg); e != nil {
			return nil, e
		}
	}

	loc.Port = port.dev.Name()
	loc.PortConfig = nil
	return New(port, loc)
}

func (loc EtherLocator) findPort() (ethdev.EthDev, *Port) {
	if loc.Port == "" {
		for _, port := range ListPorts() {
			if macaddr.Equal(port.local, loc.local()) {
				return port.dev, port
			}
		}
		for _, dev := range ethdev.List() {
			if macaddr.Equal(dev.MacAddr(), loc.local()) {
				return dev, FindPort(dev)
			}
		}
	} else {
		for _, dev := range ethdev.List() {
			if dev.Name() == loc.Port {
				return dev, FindPort(dev)
			}
		}
	}
	return ethdev.EthDev{}, nil
}

func init() {
	iface.RegisterLocatorType(EtherLocator{}, schemeEther)
}
