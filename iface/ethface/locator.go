package ethface

/*
#include <rte_ether.h>
*/
import "C"
import (
	"errors"
	"fmt"

	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/iface"
)

const locatorScheme = "ether"

// NdnMcastAddr is the well-known Ethernet multicast address for NDN traffic.
var NdnMcastAddr ethdev.EtherAddr

func init() {
	NdnMcastAddr, _ = ethdev.ParseEtherAddr("01:00:5E:00:17:AA")
}

type Locator struct {
	iface.LocatorBase
	Port   string
	Local  ethdev.EtherAddr
	Remote ethdev.EtherAddr
	Vlan   []uint16 `json:",omitempty"`
}

func NewLocator(dev ethdev.EthDev) (loc Locator) {
	loc.Scheme = locatorScheme
	loc.Port = dev.GetName()
	loc.Local = dev.GetMacAddr()
	loc.Remote = NdnMcastAddr
	return loc
}

func (loc Locator) Validate() error {
	if loc.Port == "" {
		return errors.New("Port must be non-empty")
	}
	if !loc.Local.IsZero() && !loc.Local.IsUnicast() {
		return errors.New("Local is not unicast")
	}
	if len(loc.Vlan) > 2 {
		return errors.New("too many Vlan tags")
	}
	for i, vid := range loc.Vlan {
		if vid == 0 || vid >= C.RTE_ETHER_MAX_VLAN_ID {
			return fmt.Errorf("Vlan[%d] is invalid", i)
		}
	}
	return nil
}

func init() {
	iface.RegisterLocatorType(Locator{}, locatorScheme)
}

// Create a face from locator.
// cfg is only used for initial port creation, and would be ignored if port exists.
// If cfg.Local is omitted, it is copied from loc.Local.
func Create(loc Locator, cfg PortConfig) (face *EthFace, e error) {
	if e = loc.Validate(); e != nil {
		return nil, e
	}

	dev := ethdev.Find(loc.Port)
	if !dev.IsValid() {
		return nil, errors.New("EthDev not found")
	}

	port := FindPort(dev)
	if port == nil {
		if cfg.Local.IsZero() {
			cfg.Local = loc.Local
		}
		if port, e = NewPort(dev, cfg); e != nil {
			return nil, e
		}
	}

	return New(port, loc)
}
