package ndnping

import (
	"time"

	"ndn-dpdk/iface/faceuri"
	"ndn-dpdk/ndn"
)

type TaskConfig struct {
	Face   FaceConfig
	Client *ClientConfig
	Server *ServerConfig
}

type FaceConfig struct {
	Remote *faceuri.FaceUri
	Local  *faceuri.FaceUri
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
