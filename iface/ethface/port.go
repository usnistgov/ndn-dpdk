package ethface

import (
	"errors"

	"github.com/pkg/math"
	"go.uber.org/zap"

	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
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
	// DisableRxFlow disables RxFlow implementation.
	DisableRxFlow bool `json:"disableRxFlow,omitempty"`

	// RxQueueSize is the hardware RX queue capacity.
	//
	// If this value is zero, it defaults to DefaultRxQueueSize.
	// It is also adjusted to satisfy driver requirements.
	RxQueueSize int `json:"rxQueueSize,omitempty"`

	// TxQueueSize is the hardware TX queue capacity.
	//
	// If this value is zero, it defaults to DefaultTxQueueSize.
	// It is also adjusted to satisfy driver requirements.
	TxQueueSize int `json:"txQueueSize,omitempty"`

	// MTU configures Maximum Transmission Unit (MTU) on the EthDev.
	// This excludes Ethernet headers, but includes VLAN/IP/UDP/VXLAN headers.
	// If this value is zero, the EthDev MTU remains unchanged.
	MTU int `json:"mtu,omitempty"`

	// DisableSetMTU skips setting MTU on the device.
	// Set to true only if the EthDev lacks support for setting MTU.
	DisableSetMTU bool `json:"disableSetMTU,omitempty"`
}

var portByEthDev = make(map[ethdev.EthDev]*Port)

// FindPort returns a Port associated with given EthDev.
func FindPort(ethdev ethdev.EthDev) *Port {
	return portByEthDev[ethdev]
}

// ListPorts returns a list of active Ports.
func ListPorts() (list []*Port) {
	for _, port := range portByEthDev {
		list = append(list, port)
	}
	return list
}

// Port organizes EthFaces on an EthDev.
type Port struct {
	cfg          PortConfig
	logger       *zap.Logger
	dev          ethdev.EthDev
	closeVdev    func()
	rxBouncePool *pktmbuf.Pool
	faces        map[iface.ID]*ethFace
	impl         impl
	nextImpl     int
}

// NewPort opens a Port.
func NewPort(dev ethdev.EthDev, cfg PortConfig) (port *Port, e error) {
	if cfg.MTU == 0 {
		cfg.MTU = dev.MTU()
		cfg.DisableSetMTU = true
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

	if FindPort(dev) != nil {
		return nil, errors.New("Port already exists")
	}

	port = &Port{
		cfg:    cfg,
		logger: logger.With(dev.ZapField("port")),
		dev:    dev,
		faces:  make(map[iface.ID]*ethFace),
	}
	if dev.DevInfo().DriverName() == "net_af_xdp" {
		if port.rxBouncePool, e = pktmbuf.NewPool(pktmbuf.PoolConfig{
			Capacity: cfg.RxQueueSize + iface.MaxBurstSize,
			Dataroom: math.MaxInt(pktmbuf.DefaultHeadroom+cfg.MTU, xdpMinDataroom),
		}, dev.NumaSocket()); e != nil {
			return nil, e
		}
	}

	port.logger.Debug("opening")
	portByEthDev[port.dev] = port
	return port, nil
}

// Close closes the port.
func (port *Port) Close() (e error) {
	if !port.dev.Valid() { // already closed
		return nil
	}

	if port.impl != nil {
		port.impl.Close()
		port.impl = nil
	}

	if port.dev.Valid() {
		delete(portByEthDev, port.dev)
		port.logger.Debug("closing")
		port.dev = ethdev.EthDev{}
	}

	if port.closeVdev != nil {
		port.closeVdev()
		port.closeVdev = nil
	}

	if port.rxBouncePool != nil {
		port.rxBouncePool.Close()
		port.rxBouncePool = nil
	}

	return nil
}

// Faces returns a list of active faces.
func (port *Port) Faces() (list []iface.Face) {
	for _, face := range port.faces {
		list = append(list, face.Face)
	}
	return list
}

// ImplName returns internal implementation name.
func (port *Port) ImplName() string {
	return port.impl.String()
}

// Switch to next impl.
func (port *Port) fallbackImpl() error {
	logEntry := port.logger
	if port.impl != nil {
		for _, face := range port.faces {
			face.SetDown(true)
		}

		logEntry = logEntry.With(zap.Stringer("old-impl", port.impl))
		if e := port.impl.Close(); e != nil {
			logEntry.Warn("impl close error",
				zap.Error(e),
			)
			return e
		}
	}

	if port.nextImpl >= len(impls) {
		logEntry.Warn("no feasible impl")
		return errors.New("no feasible impl")
	}
	port.impl = impls[port.nextImpl](port)
	port.nextImpl++
	logEntry = logEntry.With(zap.Stringer("impl", port.impl))

	if e := port.impl.Init(); e != nil {
		logEntry.Info("impl init error, trying next impl",
			zap.Error(e),
		)
		return port.fallbackImpl()
	}

	for faceID, face := range port.faces {
		if e := port.impl.Start(face); e != nil {
			logEntry.Info("face restart error, trying next impl",
				faceID.ZapField("face"),
				zap.Error(e),
			)
			return port.fallbackImpl()
		}
		face.SetDown(false)
	}

	logEntry.Info("impl initialized")
	return nil
}

// Start face in impl (called by New).
func (port *Port) startFace(face *ethFace, forceFallback bool) error {
	if port.impl == nil || forceFallback {
		if e := port.fallbackImpl(); e != nil {
			return e
		}
	}

	if e := port.impl.Start(face); e != nil {
		port.logger.Info("face start error, trying next impl",
			zap.Error(e),
		)
		return port.startFace(face, true)
	}

	port.logger.Info("face started",
		zap.Stringer("impl", port.impl),
		face.ID().ZapField("face"),
	)
	port.faces[face.ID()] = face
	return nil
}

// Stop face in impl (called by ethFace.Close).
func (port *Port) stopFace(face *ethFace) (e error) {
	id := face.ID()
	delete(port.faces, id)
	e = port.impl.Stop(face)
	port.logger.Info("face stopped",
		zap.Error(e),
		id.ZapField("face"),
	)
	return nil
}
