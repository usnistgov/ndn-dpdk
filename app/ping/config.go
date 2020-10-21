package ping

import (
	"github.com/pkg/math"
	"github.com/usnistgov/ndn-dpdk/app/fetch"
	"github.com/usnistgov/ndn-dpdk/app/pingclient"
	"github.com/usnistgov/ndn-dpdk/app/pingserver"
	"github.com/usnistgov/ndn-dpdk/iface"
)

// TaskConfig describes traffic generator task on a face.
type TaskConfig struct {
	Face   iface.LocatorWrapper // face locator for face creation
	Server *ServerConfig        // if not nil, create a server
	Client *pingclient.Config   // if not nil, create a client; conflicts with Fetch
	Fetch  *fetch.FetcherConfig // if not nil, create a fetcher; conflicts with Client
}

// EstimateLCores estimates how many LCores are required to activate this task.
func (cfg TaskConfig) EstimateLCores() (n int) {
	n = 2 // RX + TX

	if cfg.Server != nil {
		n += math.MaxInt(1, cfg.Server.NThreads) // SVR
	}

	if cfg.Client != nil {
		n += 2 // CLIR + CLIT
	} else if cfg.Fetch != nil {
		n++ // CLIR
	}

	return n
}

// ServerConfig describes traffic generator server task.
type ServerConfig struct {
	pingserver.Config
	NThreads int
}
