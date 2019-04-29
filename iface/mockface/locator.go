package mockface

import (
	"ndn-dpdk/iface"
)

const locatorScheme = "mock"

type Locator struct {
	iface.LocatorBase `yaml:",inline"`
}

func NewLocator() (loc Locator) {
	loc.Scheme = locatorScheme
	return loc
}

func (Locator) Validate() error {
	return nil
}

func init() {
	iface.RegisterLocatorType(Locator{}, locatorScheme)
}
