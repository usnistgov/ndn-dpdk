package ethface

/*
#include "../../csrc/ethface/eth-face.h"
*/
import "C"
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

// Port organizes EthFaces on an EthDev.
type Port struct {
	cfg      PortConfig
	logger   logrus.FieldLogger
	dev      ethdev.EthDev
	faces    map[iface.ID]*EthFace
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

	port = new(Port)
	port.cfg = cfg
	port.logger = newPortLogger(dev)
	port.dev = dev
	port.faces = make(map[iface.ID]*EthFace)

	port.logger.Debug("opening")
	portByEthDev[port.dev] = port
	return port, nil
}

// Close closes the port.
func (port *Port) Close() (e error) {
	if port.impl != nil {
		e = port.impl.Close()
	}
	delete(portByEthDev, port.dev)
	port.logger.Debug("closing")
	return nil
}

func (port *Port) findFace(filter func(face *EthFace) bool) *EthFace {
	for _, face := range port.faces {
		if filter(face) {
			return face
		}
	}
	return nil
}

// FindFace returns a face that matches the query, or nil if it does not exist.
// FindFace(nil) returns a face with multicast address.
// FindFace(unicastAddr) returns a face with matching address.
func (port *Port) FindFace(query *ethdev.EtherAddr) *EthFace {
	if query == nil {
		return port.findFace(func(face *EthFace) bool {
			return face.loc.Remote.IsGroup()
		})
	}
	return port.findFace(func(face *EthFace) bool {
		return face.loc.Remote.Equal(*query)
	})
}

// CountFaces returns the number of active faces.
func (port *Port) CountFaces() int {
	return len(port.faces)
}

// Faces returns a list of active faces.
func (port *Port) Faces() (list []*EthFace) {
	for _, face := range port.faces {
		list = append(list, face)
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
func (port *Port) startFace(face *EthFace, forceFallback bool) error {
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

// Stop face in impl (called by EthFace.Close).
func (port *Port) stopFace(face *EthFace) (e error) {
	delete(port.faces, face.ID())
	e = port.impl.Stop(face)
	port.logger.WithError(e).Info("face stopped")
	return nil
}
