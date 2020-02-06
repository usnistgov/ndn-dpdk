package ping

import (
	"ndn-dpdk/app/fetch"
	"ndn-dpdk/app/pingclient"
	"ndn-dpdk/app/pingserver"
	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

// Per-face task config, consists of a client and/or a server.
type TaskConfig struct {
	Face   iface.LocatorWrapper // face locator for face creation
	Fetch  *FetchConfig         // if not nil, create a fetcher; conflicts with Client
	Client *pingclient.Config   // if not nil, create a client; conflicts with Fetch
	Server *pingserver.Config   // if not nil, create a server
}

// Fetcher config and initial job.
type FetchConfig struct {
	fetch.FetcherConfig
	Name        *ndn.Name // if not nil, start a fetch job for this name
	FinalSegNum *uint64
}
