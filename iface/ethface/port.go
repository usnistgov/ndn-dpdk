package ethface

import (
	"errors"
	"fmt"
	"sync"

	"github.com/pkg/math"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev/ethnetif"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

// Limits and defaults.
const (
	DefaultRxQueueSize = 4096
	DefaultTxQueueSize = 4096

	xdpMinDataroom = 2048 // XDP_UMEM_MIN_CHUNK_SIZE in kernel
)

// PortConfig contains Port creation arguments.
type PortConfig struct {
	ethnetif.Config
	EthDev ethdev.EthDev `json:"-"` // override EthDev

	RxQueueSize int `json:"rxQueueSize,omitempty" gqldesc:"Hardware RX queue capacity."`
	TxQueueSize int `json:"txQueueSize,omitempty" gqldesc:"Hardware TX queue capacity."`

	MTU int `json:"mtu,omitempty" gqldesc:"Change interface MTU (excluding Ethernet/VLAN headers)."`

	RxFlowQueues int `json:"rxFlowQueues,omitempty" gqldesc:"Enable RxFlow and set maximum queue count."`
}

var (
	portByEthDev      = map[ethdev.EthDev]*Port{}
	closeAllPortsOnce sync.Once
)

// Port organizes EthFaces on an EthDev.
type Port struct {
	cfg          PortConfig
	logger       *zap.Logger
	dev          ethdev.EthDev
	devInfo      ethdev.DevInfo
	faces        map[iface.ID]*ethFace
	rxBouncePool *pktmbuf.Pool
	rxImpl       rxImpl
	txl          iface.TxLoop
	autoClose    bool
}

// Close closes the port.
func (port *Port) Close() error {
	if nFaces := len(port.faces); nFaces > 0 {
		return fmt.Errorf("cannot close Port with %d active faces", nFaces)
	}

	errs := []error{}

	if port.rxImpl != nil {
		errs = append(errs, port.rxImpl.Close(port))
		port.rxImpl = nil
	}

	if port.dev != nil {
		if port.devInfo.IsVDev() {
			errs = append(errs, port.dev.Stop(ethdev.StopDetach))
		}
		delete(portByEthDev, port.dev)
		port.dev = nil
	}

	if port.rxBouncePool != nil {
		errs = append(errs, port.rxBouncePool.Close())
		port.rxBouncePool = nil
	}

	e := multierr.Combine(errs...)
	if e != nil {
		port.logger.Error("port closed", zap.Error(e))
	} else {
		port.logger.Info("port closed")
	}
	return e
}

// Faces returns a list of active faces.
func (port *Port) Faces() (list []iface.Face) {
	for _, face := range port.faces {
		list = append(list, face.Face)
	}
	return list
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

func (port *Port) activateTx(face *ethFace) {
	if port.txl == nil {
		port.txl = iface.ActivateTxFace(face)
	} else {
		port.txl.Add(face)
	}
}

func (port *Port) deactivateTx(face *ethFace) {
	iface.DeactivateTxFace(face)
	if len(port.faces) == 0 {
		port.txl = nil
	}
}

// NewPort opens a Port.
func NewPort(cfg PortConfig) (port *Port, e error) {
	dev := cfg.EthDev
	if dev == nil {
		dev, e = ethnetif.CreateEthDev(cfg.Config)
		if e != nil {
			return nil, e
		}
	}
	if portByEthDev[dev] != nil {
		return nil, errors.New("Port already exists")
	}

	if cfg.MTU == 0 {
		cfg.MTU = dev.MTU()
	}
	if ndni.PacketMempool.Config().Dataroom < pktmbuf.DefaultHeadroom+cfg.MTU {
		return nil, errors.New("PacketMempool dataroom is too small for requested MTU")
	}
	if cfg.RxQueueSize == 0 {
		cfg.RxQueueSize = DefaultRxQueueSize
	}
	if cfg.TxQueueSize == 0 {
		cfg.TxQueueSize = DefaultTxQueueSize
	}

	port = &Port{
		cfg:     cfg,
		dev:     dev,
		devInfo: dev.DevInfo(),
		faces:   map[iface.ID]*ethFace{},
	}
	switch port.devInfo.DriverName() {
	case ethdev.DriverXDP:
		if port.rxBouncePool, e = pktmbuf.NewPool(pktmbuf.PoolConfig{
			Capacity: cfg.RxQueueSize + iface.MaxBurstSize,
			Dataroom: math.MaxInt(pktmbuf.DefaultHeadroom+cfg.MTU, xdpMinDataroom),
		}, dev.NumaSocket()); e != nil {
			return nil, e
		}
	case ethdev.DriverMemif:
		port.rxImpl = &rxMemifImpl{}
	}
	if port.rxImpl == nil {
		if port.cfg.RxFlowQueues > 0 {
			port.rxImpl = &rxFlowImpl{}
		} else {
			port.rxImpl = &rxTableImpl{}
		}
	}

	port.logger = logger.With(dev.ZapField("port"), zap.String("rxImpl", string(port.rxImpl.Kind())))
	if e := port.rxImpl.Init(port); e != nil {
		port.logger.Error("rxImpl init error", zap.Error(e))
		port.rxImpl = nil
		port.Close()
		return nil, e
	}

	port.logger.Info("port opened")
	portByEthDev[port.dev] = port
	closeAllPortsOnce.Do(func() {
		iface.OnCloseAll(func() {
			for _, port := range portByEthDev {
				port.Close()
			}
		})
	})
	return port, nil
}

// FindPort returns Port on EthDev.
func FindPort(dev ethdev.EthDev) *Port {
	return portByEthDev[dev]
}
