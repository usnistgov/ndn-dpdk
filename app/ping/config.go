package ping

import (
	"ndn-dpdk/app/pingclient"
	"ndn-dpdk/app/pingserver"
	"ndn-dpdk/iface"
)

// Package initialization config.
type InitConfig struct {
	QueueCapacity int // input-client/server queue capacity, must be power of 2
}

// Per-face task config, consists of a client and/or a server.
type TaskConfig struct {
	Face   iface.LocatorWrapper // face locator for face creation
	Client *pingclient.Config   // if not nil, create a client on the face
	Server *pingserver.Config   // if not nil, create a server on the face
}
