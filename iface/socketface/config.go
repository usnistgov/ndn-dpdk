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
	// Socket chooses a NUMA socket to create RX/TX threads for socket faces.
	// Default is the first NUMA socket.
	// If the specified NUMA socket does not exist, it uses the default.
	Socket eal.NumaSocket `json:"socket"`

	// RxConns configures net.Conn RX implementation.
	RxConns struct {
		// RingCapacity is the capacity of a ring buffer for packets that are received from net.Conn
		// but have not been picked up by C code.
		RingCapacity int `json:"ringCapacity"`
	} `json:"rxConns"`

	// RxEpoll configures epoll RX implementation, available for UDP sockets only.
	// If this is disabled, UDP sockets will use net.Conn RX implementation.
	RxEpoll struct {
		Disabled bool `json:"disabled"`
	} `json:"rxEpoll"`

	// TxSyscall configures syscall TX implementation, available for UDP sockets only.
	// If this is disabled, UDP sockets will use net.Conn TX implementation.
	TxSyscall struct {
		Disabled bool `json:"disabled"`
	} `json:"txSyscall"`
}

func (cfg GlobalConfig) Apply() {
	if !slices.Contains(eal.Sockets, cfg.Socket) {
		cfg.Socket = eal.NumaSocket{}
	}
	cfg.RxConns.RingCapacity = ringbuffer.AlignCapacity(cfg.RxConns.RingCapacity, 64, 4096, 65536)
	gCfg = cfg
}

func (cfg GlobalConfig) numaSocket() eal.NumaSocket {
	return eal.RewriteAnyNumaSocketFirst.Rewrite(cfg.Socket)
}

var gCfg GlobalConfig

func init() {
	GlobalConfig{}.Apply()
}
