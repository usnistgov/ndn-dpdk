package ethface

import (
	"errors"
	"net"

	"ndn-dpdk/iface"
)

const locatorScheme = "ether"

type Locator struct {
	iface.LocatorBase `yaml:",inline"`
	Port              string
	Local             net.HardwareAddr
	Remote            net.HardwareAddr
}

func (loc Locator) Validate() error {
	if loc.Port == "" {
		return errors.New("Port must be non-empty")
	}
	if len(loc.Local) != 6 || (loc.Local[0]&0x01) != 0 {
		return errors.New("Local must be MAC-48 unicast address")
	}
	if len(loc.Remote) != 6 {
		return errors.New("Remote must be MAC-48 address")
	}
	return nil
}

func (loc Locator) IsRemoteMulticast() bool {
	return len(loc.Remote) == 6 && (loc.Remote[0]&0x01) != 0
}

func init() {
	iface.RegisterLocatorType(Locator{}, "ether")
}
