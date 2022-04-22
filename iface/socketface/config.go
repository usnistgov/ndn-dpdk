package socketface

import (
	"github.com/usnistgov/ndn-dpdk/core/nnduration"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ringbuffer"
	"github.com/usnistgov/ndn-dpdk/iface"
	"golang.org/x/exp/slices"
)

// Config contains socket face configuration.
type Config struct {
	iface.Config

	// sockettransport.Config fields.
	// See ndn-dpdk/ndn/sockettransport package for their semantics and defaults.
	RedialBackoffInitial nnduration.Milliseconds `json:"redialBackoffInitial,omitempty"`
	RedialBackoffMaximum nnduration.Milliseconds `json:"redialBackoffMaximum,omitempty"`
}

// GlobalConfig contains global options applied to all socket faces.
type GlobalConfig struct {
	RxConns struct {
		Disabled     bool           `json:"disabled"`
		RingCapacity int            `json:"ringCapacity"`
		Socket       eal.NumaSocket `json:"socket"`
	} `json:"rxConns"`
	RxEpoll struct {
		Disabled bool           `json:"disabled"`
		Socket   eal.NumaSocket `json:"socket"`
	} `json:"rxEpoll"`
}

var gCfg GlobalConfig

func (cfg GlobalConfig) Apply() {
	cfg.RxConns.RingCapacity = ringbuffer.AlignCapacity(cfg.RxConns.RingCapacity, 64, 4096, 65536)
	if !slices.Contains(eal.Sockets, cfg.RxConns.Socket) {
		cfg.RxConns.Socket = eal.NumaSocket{}
	}
	if !slices.Contains(eal.Sockets, cfg.RxEpoll.Socket) {
		cfg.RxEpoll.Socket = eal.NumaSocket{}
	}
	gCfg = cfg
}

func init() {
	GlobalConfig{}.Apply()
}
