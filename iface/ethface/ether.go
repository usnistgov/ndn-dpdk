package ethface

import (
	"errors"

	"github.com/pkg/math"
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

// EtherLocatorInput contains input-only fields of EtherLocator.
type EtherLocatorInput struct {
	// PortConfig specifies additional configuration for Port activation.
	// This is only used when creating the first face on an EthDev.
	PortConfig *PortConfig `json:"portConfig,omitempty"`

	// MaxRxQueues is the maximum number of RX queues for this face.
	// It is meaningful only if the face is using RxFlow dispatching.
	// It is effective in improving performance on VXLAN face only.
	//
	// Default is 1.
	// If this is greater than 1, NDNLPv2 reassembly will not work on this face.
	MaxRxQueues int `json:"maxRxQueues,omitempty"`

	// privInput is hidden from JSON output.
	privInput *EtherLocatorInput
}

func (loc EtherLocatorInput) maxRxQueues() int {
	if loc.privInput != nil {
		return loc.privInput.maxRxQueues()
	}
	return math.MaxInt(loc.MaxRxQueues, 1)
}

// EtherLocator describes an Ethernet face.
type EtherLocator struct {
	EtherLocatorInput

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
	if !dev.Valid() {
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
	input := loc.EtherLocatorInput
	input.privInput = nil
	loc.EtherLocatorInput = EtherLocatorInput{privInput: &input}
	return port, nil
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
