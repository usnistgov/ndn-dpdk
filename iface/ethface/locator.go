package ethface

/*
#include <rte_ether.h>
*/
import "C"
import (
	"encoding/json"
	"errors"

	"github.com/VojtechVitek/mergemaps"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn/packettransport"
)

const locatorScheme = "ether"

// LocatorFields contains additional Locator fields.
type LocatorFields struct {
	Port string `json:"port"`
}

// Locator describes port, addresses, and VLAN of an Ethernet face.
type Locator struct {
	packettransport.Locator
	LocatorFields
}

// NewLocator creates a Locator.
func NewLocator(dev ethdev.EthDev) (loc Locator) {
	loc.Port = dev.Name()
	loc.Local = dev.MacAddr()
	loc.Remote = packettransport.MulticastAddressNDN
	return loc
}

// Scheme returns "ether".
func (Locator) Scheme() string {
	return locatorScheme
}

// Validate checks Locator fields.
func (loc Locator) Validate() error {
	if loc.Port == "" {
		return errors.New("invalid Port")
	}
	return loc.Locator.Validate()
}

// MarshalJSON implements json.Marshaler.
func (loc Locator) MarshalJSON() (data []byte, e error) {
	dst := map[string]string{
		"scheme": locatorScheme,
	}
	var src map[string]string

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

// Create creates a face from locator.
// cfg is only used for initial port creation, and would be ignored if port exists.
// If cfg.Local is omitted, it is copied from loc.Local.
func Create(loc Locator, cfg PortConfig) (face iface.Face, e error) {
	if e = loc.Validate(); e != nil {
		return nil, e
	}

	dev := ethdev.Find(loc.Port)
	if !dev.Valid() {
		return nil, errors.New("EthDev not found")
	}

	port := FindPort(dev)
	if port == nil {
		if cfg.Local == nil {
			cfg.Local = loc.Local
		}
		if port, e = NewPort(dev, cfg); e != nil {
			return nil, e
		}
	}

	return New(port, loc)
}
