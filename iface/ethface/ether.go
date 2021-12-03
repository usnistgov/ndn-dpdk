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
	// During face creation, Ethernet adapters are searched in this order:
	// 1. Existing EthDevs are considered first, including:
	//    * Physical Ethernet adapters using DPDK PCI drivers.
	//    * Virtual EthDev created during prior face creations.
	// 2. Kernel-managed network interfaces are considered next.
	//    They will be activated as virtual EthDev using net_af_xdp or net_af_packet driver.
	//    loc.VDevConfig contains parameters for virtual device creation.
	//
	// If this is empty:
	// * Face is created on an Ethernet adapter whose physical MAC address equals loc.Local.
	// * Local MAC address of the face is set to loc.Local, i.e. same as the physical MAC address.
	//
	// If this is non-empty:
	// * Face is created on an Ethernet adapter whose name equals loc.Port.
	//   * loc.Port can be either the kernel network interface name or its PCI address.
	//   * If the PCI device is bound to a generic driver (e.g. uio_pci_generic), loc.Port can only be
	//     a PCI address, because the kernel no longer recognizes its as a network interface.
	// * Local MAC address of the face is set to loc.Local, which could differ from the physical MAC address.
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
