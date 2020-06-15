package ping

import (
	"github.com/usnistgov/ndn-dpdk/app/fetch"
	"github.com/usnistgov/ndn-dpdk/app/pingclient"
	"github.com/usnistgov/ndn-dpdk/app/pingserver"
	"github.com/usnistgov/ndn-dpdk/iface"
)

// Per-face task config.
type TaskConfig struct {
	Face   iface.LocatorWrapper // face locator for face creation
	Server *ServerConfig        // if not nil, create a server
	Client *pingclient.Config   // if not nil, create a client; conflicts with Fetch
	Fetch  *fetch.FetcherConfig // if not nil, create a fetcher; conflicts with Client
}

type ServerConfig struct {
	pingserver.Config
	NThreads int
}
