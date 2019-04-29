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
	iface.RegisterLocatorType(Locator{}, "ether")
}
