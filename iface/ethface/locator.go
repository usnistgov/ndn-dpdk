package ethface

/*
#include <rte_ether.h>
*/
import "C"
import (
	"errors"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

const locatorScheme = "ether"

type Locator struct {
	iface.LocatorBase
	Port   string
	Local  dpdk.EtherAddr
	Remote dpdk.EtherAddr
}

func NewLocator(dev dpdk.EthDev) (loc Locator) {
	loc.Scheme = locatorScheme
	loc.Port = dev.GetName()
	loc.Local = dev.GetMacAddr()
	loc.Remote = ndn.NDN_ETHER_MCAST_ADDR
	return loc
}

func (loc Locator) Validate() error {
	if loc.Port == "" {
		return errors.New("Port must be non-empty")
	}
	if !loc.Local.IsZero() && !loc.Local.IsUnicast() {
		return errors.New("Local is not unicast")
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

	dev := dpdk.FindEthDev(loc.Port)
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
