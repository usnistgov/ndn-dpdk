package mockface

import (
	"ndn-dpdk/iface"
)

const locatorScheme = "mock"

type Locator struct {
	iface.LocatorBase `yaml:",inline"`
}

func (Locator) Validate() error {
	return nil
}

func init() {
	iface.RegisterLocatorType(Locator{}, locatorScheme)
}
