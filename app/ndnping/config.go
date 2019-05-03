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
	Prefix           *ndn.Name
	CanBePrefix      bool
	MustBeFresh      bool
	InterestLifetime time.Duration
	HopLimit         int
}

type ServerConfig struct {
	Patterns []ServerPattern
	Nack     bool
}

type ServerPattern struct {
	Prefix          *ndn.Name
	Suffix          *ndn.Name
	FreshnessPeriod time.Duration
	PayloadLen      int
}
