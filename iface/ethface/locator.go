package ethface

import (
	"encoding/json"
	"errors"

	"github.com/VojtechVitek/mergemaps"
	"github.com/usnistgov/ndn-dpdk/core/macaddr"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn/packettransport"
)

const locatorScheme = "ether"

// LocatorFields contains additional Locator fields.
type LocatorFields struct {
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

// Scheme returns "ether".
func (Locator) Scheme() string {
	return locatorScheme
}

// CreateFace creates a face from this Locator.
func (loc Locator) CreateFace() (face iface.Face, e error) {
	if e = loc.Validate(); e != nil {
		return nil, e
	}
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

// MarshalJSON implements json.Marshaler.
func (loc Locator) MarshalJSON() (data []byte, e error) {
	dst := map[string]interface{}{
		"scheme": locatorScheme,
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
	iface.RegisterLocatorType(Locator{}, locatorScheme)
}
