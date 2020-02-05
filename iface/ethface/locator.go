package ethface

/*
#include <rte_ether.h>
*/
import "C"
import (
	"encoding/json"
	"errors"
	"net"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

type classifyMac48Result int

const (
	mac48_no classifyMac48Result = iota
	mac48_unicast
	mac48_multicast
)

func classifyMac48(addr net.HardwareAddr) classifyMac48Result {
	switch {
	case len(addr) != 6:
		return mac48_no
	case (addr[0] & 0x01) == 1:
		return mac48_multicast
	}
	return mac48_unicast
}

func copyMac48ToC(a net.HardwareAddr, c *C.struct_rte_ether_addr) {
	for i := 0; i < C.RTE_ETHER_ADDR_LEN; i++ {
		c.addr_bytes[i] = C.uint8_t(a[i])
	}
}

const locatorScheme = "ether"

type Locator struct {
	iface.LocatorBase
	Port   string
	Local  net.HardwareAddr
	Remote net.HardwareAddr
}

func NewLocator(dev dpdk.EthDev) (loc Locator) {
	loc.Scheme = locatorScheme
	loc.Port = dev.GetName()
	loc.Local = dev.GetMacAddr()
	loc.Remote = ndn.GetEtherMcastAddr()
	return loc
}

func (loc Locator) Validate() error {
	if loc.Port == "" {
		return errors.New("Port must be non-empty")
	}
	if loc.Local != nil && classifyMac48(loc.Local) != mac48_unicast {
		return errors.New("Local must be MAC-48 unicast address")
	}
	if loc.Remote != nil && classifyMac48(loc.Remote) == mac48_no {
		return errors.New("Remote must be MAC-48 address")
	}
	return nil
}

func (loc Locator) IsRemoteMulticast() bool {
	return classifyMac48(loc.Remote) == mac48_multicast
}

type locatorJson struct {
	iface.LocatorBase
	Port   string
	Local  string
	Remote string
}

func (loc Locator) MarshalJSON() ([]byte, error) {
	var output locatorJson
	output.LocatorBase = loc.LocatorBase
	output.Port = loc.Port
	output.Local = loc.Local.String()
	output.Remote = loc.Remote.String()
	return json.Marshal(output)
}

func (loc *Locator) UnmarshalJSON(data []byte) (e error) {
	var input locatorJson
	if e = json.Unmarshal(data, &input); e != nil {
		return e
	}
	loc.LocatorBase = input.LocatorBase
	loc.Port = input.Port
	if input.Local == "" {
		loc.Local = nil
	} else if loc.Local, e = net.ParseMAC(input.Local); e != nil {
		return e
	}
	if input.Remote == "" {
		loc.Remote = nil
	} else if loc.Remote, e = net.ParseMAC(input.Remote); e != nil {
		return e
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
		if cfg.Local == nil {
			cfg.Local = loc.Local
		}
		if port, e = NewPort(dev, cfg); e != nil {
			return nil, e
		}
	}

	return New(port, loc)
}
