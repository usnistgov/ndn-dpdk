package ndnping

import (
	"time"

	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

type TaskConfig struct {
	Face   iface.LocatorWrapper
	Client *ClientConfig
	Server *ServerConfig
}

type ClientConfig struct {
	Patterns []ClientPattern
	Interval time.Duration
}

type ClientPattern struct {
	Prefix *ndn.Name
}

func (pattern ClientPattern) AsInterestTemplate() (tpl *ndn.InterestTemplate) {
	tpl = ndn.NewInterestTemplate()
	tpl.SetNamePrefix(pattern.Prefix)
	tpl.SetCanBePrefix(true)
	tpl.SetMustBeFresh(true)
	tpl.SetInterestLifetime(1000 * time.Millisecond)
	tpl.SetHopLimit(255)
	return tpl
}

type ServerConfig struct {
	Patterns []ServerPattern
	Nack     bool
}

type ServerPattern struct {
	Prefix     *ndn.Name
	PayloadLen int
	Suffix     *ndn.Name
}
