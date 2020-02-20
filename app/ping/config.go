package ping

import (
	"ndn-dpdk/app/fetch"
	"ndn-dpdk/app/pingclient"
	"ndn-dpdk/app/pingserver"
	"ndn-dpdk/iface"
)

// Per-face task config.
type TaskConfig struct {
	Face     iface.LocatorWrapper // face locator for face creation
	Server   *pingserver.Config   // if not nil, create a server
	Client   *pingclient.Config   // if not nil, create a client; conflicts with Fetch
	Fetch    int                  // number of fetchers; conflicts with Client
	FetchCfg fetch.FetcherConfig  // fetcher options
}
