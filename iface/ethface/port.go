package ethface

import (
	"errors"

	"github.com/sirupsen/logrus"

	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/iface"
)

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

// PortConfig contains Port creation arguments.
type PortConfig struct {
	RxqFrames  int              // RX queue capacity
	TxqPkts    int              // before-TX queue capacity
	TxqFrames  int              // after-TX queue capacity
	Mtu        int              // set MTU, 0 to retrieve from EthDev (implies SkipSetMtu)
	SkipSetMtu bool             // if true, don't set MTU on EthDev; MTU is only used for fragmentation
	Local      ethdev.EtherAddr // local address, zero for hardware default
}

func (cfg PortConfig) check() error {
	if !cfg.Local.IsZero() && !cfg.Local.IsUnicast() {
		return errors.New("Local is not unicast")
	}
	return nil
}

// Port organizes EthFaces on an EthDev.
type Port struct {
	cfg      PortConfig
	logger   logrus.FieldLogger
	dev      ethdev.EthDev
	faces    map[iface.ID]*ethFace
	impl     impl
	nextImpl int
}

// NewPort opens a Port.
func NewPort(dev ethdev.EthDev, cfg PortConfig) (port *Port, e error) {
	if e = cfg.check(); e != nil {
		return nil, e
	}
	if FindPort(dev) != nil {
		return nil, errors.New("Port already exists")
	}
	if cfg.Local.IsZero() {
		cfg.Local = dev.MacAddr()
	}
	if cfg.Mtu == 0 {
		cfg.Mtu = dev.Mtu()
		cfg.SkipSetMtu = true
	}

	port = new(Port)
	port.cfg = cfg
	port.logger = newPortLogger(dev)
	port.dev = dev
	port.faces = make(map[iface.ID]*ethFace)

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
	return nil
}

func (port *Port) filterFace(filter func(face *ethFace) bool) iface.Face {
	for _, face := range port.faces {
		if filter(face) {
			return face.Face
		}
	}
	return nil
}

// FindFace returns a face that matches the query, or nil if it does not exist.
// FindFace(nil) returns a face with multicast address.
// FindFace(unicastAddr) returns a face with matching address.
func (port *Port) FindFace(query *ethdev.EtherAddr) iface.Face {
	if query == nil {
		return port.filterFace(func(face *ethFace) bool {
			return face.loc.Remote.IsGroup()
		})
	}
	return port.filterFace(func(face *ethFace) bool {
		return face.loc.Remote.Equal(*query)
	})
}

// CountFaces returns the number of active faces.
func (port *Port) CountFaces() int {
	return len(port.faces)
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

		logEntry = logEntry.WithField("old-impl", port.impl.String())
		if e := port.impl.Close(); e != nil {
			logEntry.WithError(e).Warn("impl close error")
			return e
		}
	}

	if port.nextImpl >= len(impls) {
		logEntry.Warn("no feasible impl")
		return errors.New("no feasible impl")
	}
	port.impl = impls[port.nextImpl](port)
	port.nextImpl++
	logEntry = logEntry.WithField("impl", port.impl.String())

	if e := port.impl.Init(); e != nil {
		logEntry.WithError(e).Info("impl init error, trying next impl")
		return port.fallbackImpl()
	}

	for faceID, face := range port.faces {
		if e := port.impl.Start(face); e != nil {
			logEntry.WithField("face", faceID).WithError(e).Info("face restart error, trying next impl")
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
		port.logger.WithError(e).Info("face start error, trying next impl")
		return port.startFace(face, true)
	}

	port.logger.WithFields(makeLogFields("impl", port.impl.String(), "face", face.ID())).Info("face started")
	port.faces[face.ID()] = face
	return nil
}

// Stop face in impl (called by ethFace.Close).
func (port *Port) stopFace(face *ethFace) (e error) {
	delete(port.faces, face.ID())
	e = port.impl.Stop(face)
	port.logger.WithError(e).Info("face stopped")
	return nil
}
