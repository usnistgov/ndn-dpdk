// Package ethport implements faces using DPDK Ethernet device as transport.
package ethport

import (
	"errors"
	"fmt"
	"sync"

	"github.com/zyedidia/generic"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"github.com/usnistgov/ndn-dpdk/core/logging"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev/ethnetif"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

var logger = logging.New("ethport")
var (
	ports      = map[ethdev.EthDev]*Port{}
	portsMutex sync.RWMutex
)

// Limits and defaults.
const (
	DefaultRxQueueSize = 4096
	DefaultTxQueueSize = 4096

	xdpMinDataroom = 2048 // XDP_UMEM_MIN_CHUNK_SIZE in kernel
)

// Config contains Port creation arguments.
type Config struct {
	// ethnetif.Config specifies how to find or create EthDev.
	ethnetif.Config
	// EthDev specifies EthDev. It overrides ethnetif.Config.
	EthDev ethdev.EthDev `json:"-"`
	// AutoClose indicates that EthDev should be closed when the last face is closed.
	AutoClose bool `json:"-"`

	RxQueueSize int `json:"rxQueueSize,omitempty" gqldesc:"Hardware RX queue capacity."`
	TxQueueSize int `json:"txQueueSize,omitempty" gqldesc:"Hardware TX queue capacity."`

	MTU int `json:"mtu,omitempty" gqldesc:"Change interface MTU (excluding Ethernet/VLAN headers)."`

	RxFlowQueues int `json:"rxFlowQueues,omitempty" gqldesc:"Enable RxFlow and set maximum queue count."`
}

// ensureEthDev creates EthDev if it's not set.
func (cfg *Config) ensureEthDev() (e error) {
	if cfg.EthDev != nil {
		return nil
	}
	if cfg.EthDev, e = ethnetif.CreateEthDev(cfg.Config); e != nil {
		return e
	}
	return nil
}

// applyDefaults applies defaults.
// cfg.EthDev must be set before calling this function.
func (cfg *Config) applyDefaults() {
	if cfg.MTU == 0 {
		cfg.MTU = cfg.EthDev.MTU()
	}
	if cfg.RxQueueSize == 0 {
		cfg.RxQueueSize = DefaultRxQueueSize
	}
	if cfg.TxQueueSize == 0 {
		cfg.TxQueueSize = DefaultTxQueueSize
	}
}

// Port organizes EthFaces on an EthDev.
type Port struct {
	mutex        sync.Mutex
	cfg          Config
	logger       *zap.Logger
	dev          ethdev.EthDev
	devInfo      ethdev.DevInfo
	faces        map[iface.ID]*Face
	rxBouncePool *pktmbuf.Pool
	rxImpl       rxImpl
	txl          iface.TxLoop
}

// EthDev returns the Ethernet device.
func (port *Port) EthDev() ethdev.EthDev {
	return port.dev
}

// Faces returns a list of active faces.
func (port *Port) Faces() (list []iface.Face) {
	port.mutex.Lock()
	defer port.mutex.Unlock()

	for _, face := range port.faces {
		list = append(list, face)
	}
	return list
}

// Close closes the port.
func (port *Port) Close() error {
	portsMutex.Lock()
	defer portsMutex.Unlock()
	return port.closeWithPortsMutex()
}

func (port *Port) closeWithPortsMutex() error {
	port.mutex.Lock()
	defer port.mutex.Unlock()

	if nFaces := len(port.faces); nFaces > 0 {
		return fmt.Errorf("cannot close Port with %d active faces", nFaces)
	}

	errs := []error{}

	if port.rxImpl != nil {
		errs = append(errs, port.rxImpl.Close(port))
		port.rxImpl = nil
	}

	if port.dev != nil {
		errs = append(errs, port.dev.Close())
		delete(ports, port.dev)
		port.dev = nil
	}

	if port.rxBouncePool != nil {
		errs = append(errs, port.rxBouncePool.Close())
		port.rxBouncePool = nil
	}

	if e := multierr.Combine(errs...); e != nil {
		port.logger.Warn("port closed with errors", zap.Error(e))
	} else {
		port.logger.Info("port closed")
	}
	return nil
}

func (port *Port) startDev(nRxQueues int, promisc bool) error {
	socket := port.dev.NumaSocket()
	rxPool := port.rxBouncePool
	if rxPool == nil {
		rxPool = ndni.PacketMempool.Get(socket)
	}

	cfg := ethdev.Config{
		MTU:     port.cfg.MTU,
		Promisc: promisc,
	}
	cfg.AddRxQueues(nRxQueues, ethdev.RxQueueConfig{
		Capacity: port.cfg.RxQueueSize,
		Socket:   socket,
		RxPool:   rxPool,
	})
	cfg.AddTxQueues(1, ethdev.TxQueueConfig{
		Capacity: port.cfg.TxQueueSize,
		Socket:   socket,
	})
	return port.dev.Start(cfg)
}

func (port *Port) activateTx(face iface.Face) {
	if port.txl == nil {
		port.txl = iface.ActivateTxFace(face)
	} else {
		port.txl.Add(face)
	}
}

func (port *Port) deactivateTx(face iface.Face) {
	iface.DeactivateTxFace(face)
	if len(port.faces) == 0 {
		port.txl = nil
	}
}

// New opens a Port.
func New(cfg Config) (port *Port, e error) {
	portsMutex.Lock()
	defer portsMutex.Unlock()
	if e = cfg.ensureEthDev(); e != nil {
		return nil, e
	}
	if ports[cfg.EthDev] != nil {
		return nil, errors.New("Port already exists")
	}

	cfg.applyDefaults()
	if ndni.PacketMempool.Config().Dataroom < pktmbuf.DefaultHeadroom+cfg.MTU {
		return nil, errors.New("PacketMempool dataroom is too small for requested MTU")
	}

	port = &Port{
		cfg:     cfg,
		logger:  logger.With(cfg.EthDev.ZapField("port")),
		dev:     cfg.EthDev,
		devInfo: cfg.EthDev.DevInfo(),
		faces:   map[iface.ID]*Face{},
	}
	switch port.devInfo.Driver() {
	case ethdev.DriverXDP:
		if port.rxBouncePool, e = pktmbuf.NewPool(pktmbuf.PoolConfig{
			Capacity: cfg.RxQueueSize + iface.MaxBurstSize,
			Dataroom: generic.Max(pktmbuf.DefaultHeadroom+cfg.MTU, xdpMinDataroom),
		}, cfg.EthDev.NumaSocket()); e != nil {
			return nil, e
		}
	case ethdev.DriverMemif:
		port.rxImpl = &rxMemif{}
	}
	if port.rxImpl == nil {
		if port.cfg.RxFlowQueues > 0 {
			port.rxImpl = &rxFlow{}
		} else {
			port.rxImpl = &rxTable{}
		}
	}

	if e := port.rxImpl.Init(port); e != nil {
		port.logger.Error("rxImpl init error", zap.Error(e))
		port.rxImpl = nil
		port.Close()
		return nil, e
	}

	port.logger.Info("port opened", zap.Stringer("rxImpl", port.rxImpl))
	ports[port.dev] = port
	return port, nil
}

// Find finds Port by EthDev.
func Find(dev ethdev.EthDev) *Port {
	if dev == nil {
		return nil
	}
	portsMutex.RLock()
	defer portsMutex.RUnlock()
	return ports[dev]
}

func init() {
	iface.OnCloseAll(func() {
		portsMutex.Lock()
		defer portsMutex.Unlock()
		for _, port := range ports {
			port.closeWithPortsMutex()
		}
	})
}
