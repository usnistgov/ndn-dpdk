package ethface

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/VojtechVitek/mergemaps"
	"github.com/usnistgov/ndn-dpdk/core/macaddr"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn/memiftransport"
	"github.com/usnistgov/ndn-dpdk/ndn/packettransport"
)

const (
	locatorSchemeEther = "ether"
	locatorSchemeMemif = "memif"
)

// LocatorFields contains additional Locator fields.
type LocatorFields struct {
	// Memif specifies shared memory packet interface (memif) settings.
	// If this is specified:
	// - loc.Scheme() becomes "memif".
	// - loc.Local is overridden as memiftransport.AddressDPDK.
	// - loc.Remote is overridden as memiftransport.AddressApp.
	// - loc.VLAN is overridden as 0.
	// - loc.Port is ignored.
	// - loc.PortConfig.MTU is overridden as loc.Memif.Dataroom.
	// - loc.PortConfig.NoSetMTU is overridden to true.
	Memif *memiftransport.Locator `json:"memif,omitempty"`

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

	// RxQueueIDs is a list of hardware RX queue numbers used by this face, if known.
	// This is meaningful on an existing face, and ignored during face creation.
	RxQueueIDs []int `json:"rxQueueIDs,omitempty"`
}

// Locator describes port, addresses, and VLAN of an Ethernet face.
type Locator struct {
	packettransport.Locator
	LocatorFields
}

// NewLocator creates a Locator.
func NewLocator(dev ethdev.EthDev) (loc Locator) {
	loc.Local = dev.MacAddr()
	loc.Remote = packettransport.MulticastAddressNDN
	return loc
}

// Scheme returns either "ether" or "memif".
func (loc Locator) Scheme() string {
	if loc.Memif != nil {
		return locatorSchemeMemif
	}
	return locatorSchemeEther
}

// CreateFace creates a face from this Locator.
func (loc Locator) CreateFace() (face iface.Face, e error) {
	if e = loc.Validate(); e != nil {
		return nil, e
	}
	if loc.Memif != nil {
		return loc.createMemif()
	}
	return loc.createEther()
}

func (loc Locator) createEther() (face iface.Face, e error) {
	dev, port := loc.findPort()
	if !dev.Valid() {
		return nil, errors.New("EthDev not found")
	}

	if port == nil {
		var cfg PortConfig
		if loc.PortConfig != nil {
			cfg = *loc.PortConfig
		}
		if port, e = NewPort(dev, loc.Local, cfg); e != nil {
			return nil, e
		}
	}

	return New(port, loc)
}

func (loc Locator) findPort() (ethdev.EthDev, *Port) {
	if loc.Port == "" {
		for _, port := range ListPorts() {
			if macaddr.Equal(port.local, loc.Local) {
				return port.dev, port
			}
		}
		for _, dev := range ethdev.List() {
			if macaddr.Equal(dev.MacAddr(), loc.Local) {
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

func (loc Locator) createMemif() (face iface.Face, e error) {
	name := "net_memif" + eal.AllocObjectID("ethface.Memif")
	args, e := loc.Memif.ToVDevArgs()
	if e != nil {
		return nil, fmt.Errorf("Memif.ToVDevArgs %w", e)
	}
	vdev, e := eal.NewVDev(name, args, eal.NumaSocket{})
	if e != nil {
		return nil, fmt.Errorf("eal.NewVDev(%s,%s) %w", name, args, e)
	}
	loc.overrideMemif(vdev)

	port, e := NewPort(ethdev.Find(vdev.Name()), loc.Local, *loc.PortConfig)
	if e != nil {
		vdev.Close()
		return nil, fmt.Errorf("NewPort %w", e)
	}
	port.vdev = vdev
	return New(port, loc)
}

func (loc *Locator) overrideMemif(vdev *eal.VDev) {
	loc.Local = memiftransport.AddressDPDK
	loc.Remote = memiftransport.AddressApp
	loc.VLAN = 0
	loc.Port = vdev.Name()
	if loc.PortConfig == nil {
		loc.PortConfig = &PortConfig{}
	}
	loc.PortConfig.MTU = loc.Memif.Dataroom
	loc.PortConfig.NoSetMTU = true
}

// MarshalJSON implements json.Marshaler.
func (loc Locator) MarshalJSON() (data []byte, e error) {
	dst := map[string]interface{}{
		"scheme": loc.Scheme(),
	}
	var src map[string]interface{}

	if data, e = json.Marshal(loc.Locator); e != nil {
		return nil, e
	}
	if e = json.Unmarshal(data, &src); e != nil {
		return nil, e
	}
	mergemaps.MergeInto(dst, src, 0)

	if data, e = json.Marshal(loc.LocatorFields); e != nil {
		return nil, e
	}
	if e = json.Unmarshal(data, &src); e != nil {
		return nil, e
	}
	mergemaps.MergeInto(dst, src, 0)

	return json.Marshal(dst)
}

// UnmarshalJSON implements json.Unmarshaler.
func (loc *Locator) UnmarshalJSON(data []byte) error {
	if e := json.Unmarshal(data, &loc.Locator); e != nil {
		return e
	}
	return json.Unmarshal(data, &loc.LocatorFields)
}

func init() {
	iface.RegisterLocatorType(Locator{}, locatorSchemeEther, locatorSchemeMemif)
}
