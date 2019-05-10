package ethface

/*
#include <rte_ether.h>
*/
import "C"
import (
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

func copyMac48ToC(a net.HardwareAddr, c *C.struct_ether_addr) {
	for i := 0; i < C.ETHER_ADDR_LEN; i++ {
		c.addr_bytes[i] = C.uint8_t(a[i])
	}
}

const locatorScheme = "ether"

type Locator struct {
	iface.LocatorBase `yaml:",inline"`
	Port              string
	Local             net.HardwareAddr
	Remote            net.HardwareAddr
}

func NewLocator(ethdev dpdk.EthDev) (loc Locator) {
	loc.Scheme = locatorScheme
	loc.Port = ethdev.GetName()
	loc.Local = ethdev.GetMacAddr()
	loc.Remote = ndn.GetEtherMcastAddr()
	return loc
}

func (loc Locator) Validate() error {
	if loc.Port == "" {
		return errors.New("Port must be non-empty")
	}
	if classifyMac48(loc.Local) != mac48_unicast {
		return errors.New("Local must be MAC-48 unicast address")
	}
	if classifyMac48(loc.Remote) == mac48_no {
		return errors.New("Remote must be MAC-48 address")
	}
	return nil
}

func (loc Locator) IsRemoteMulticast() bool {
	return classifyMac48(loc.Remote) == mac48_multicast
}

type locatorYaml struct {
	iface.LocatorBase `yaml:",inline"`
	Port              string
	Local             string
	Remote            string
}

func (loc Locator) MarshalYAML() (interface{}, error) {
	var output locatorYaml
	output.LocatorBase = loc.LocatorBase
	output.Port = loc.Port
	output.Local = loc.Local.String()
	output.Remote = loc.Remote.String()
	return output, nil
}

func (loc *Locator) UnmarshalYAML(unmarshal func(interface{}) error) (e error) {
	var input locatorYaml
	if e = unmarshal(&input); e != nil {
		return e
	}
	loc.LocatorBase = input.LocatorBase
	loc.Port = input.Port
	if loc.Local, e = net.ParseMAC(input.Local); e != nil {
		return e
	}
	if loc.Remote, e = net.ParseMAC(input.Remote); e != nil {
		return e
	}
	return nil
}

func init() {
	iface.RegisterLocatorType(Locator{}, locatorScheme)
}
