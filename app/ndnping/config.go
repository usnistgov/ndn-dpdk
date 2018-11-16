package ndnping

import (
	"time"

	"ndn-dpdk/iface/createface"
	"ndn-dpdk/ndn"
)

type TaskConfig struct {
	Face   createface.CreateArg
	Client *ClientConfig
	Server *ServerConfig
}

type ClientConfig struct {
	Patterns []ClientPattern
	Interval time.Duration
}

type ClientPattern struct {
	Prefix *ndn.Name
	Repeat int
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
