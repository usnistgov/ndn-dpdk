package ethface

/*
#include "eth-face.h"
*/
import "C"
import (
	"bytes"
	"errors"
	"net"

	"github.com/sirupsen/logrus"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
)

var portByEthDev = make(map[dpdk.EthDev]*Port)

func FindPort(ethdev dpdk.EthDev) *Port {
	return portByEthDev[ethdev]
}

func ListPorts() (list []*Port) {
	for _, port := range portByEthDev {
		list = append(list, port)
	}
	return list
}

// Collection of EthFaces on a DPDK EthDev.
type Port struct {
	cfg    PortConfig
	logger logrus.FieldLogger
	dev    dpdk.EthDev
	impl   iImpl
	faces  map[iface.FaceId]*EthFace
}

// Create a port without starting it.
func NewPort(cfg PortConfig) (port *Port, e error) {
	if e = cfg.check(); e != nil {
		return nil, e
	}
	if FindPort(cfg.EthDev) != nil {
		return nil, errors.New("cfg.EthDev matches existing Port")
	}
	if cfg.Local == nil {
		cfg.Local = port.dev.GetMacAddr()
	}

	port = new(Port)
	port.cfg = cfg
	port.logger = newPortLogger(cfg.EthDev)
	port.dev = cfg.EthDev
	port.impl = newRxFlowImpl(port)
	port.faces = make(map[iface.FaceId]*EthFace)
	if e = port.impl.Init(); e != nil {
		return nil, e
	}

	portByEthDev[cfg.EthDev] = port
	return port, nil
}

func (port *Port) GetEthDev() dpdk.EthDev {
	return port.dev
}

func (port *Port) Close() error {
	if e := port.impl.Close(); e != nil {
		return e
	}
	delete(portByEthDev, port.dev)
	port.logger.Debug("closing")
	return nil
}

func (port *Port) startDev(nRxQueues int, promisc bool) error {
	var cfg dpdk.EthDevConfig
	numaSocket := port.dev.GetNumaSocket()
	for i := 0; i < nRxQueues; i++ {
		cfg.AddRxQueue(dpdk.EthRxQueueConfig{
			Capacity: port.cfg.RxqFrames,
			Socket:   numaSocket,
			Mp:       port.cfg.RxMp,
		})
	}
	cfg.AddTxQueue(dpdk.EthTxQueueConfig{
		Capacity: port.cfg.TxqFrames,
		Socket:   numaSocket,
	})
	cfg.Mtu = port.cfg.Mtu
	if _, _, e := port.dev.Configure(cfg); e != nil {
		return e
	}

	port.dev.SetPromiscuous(promisc)
	return port.dev.Start()
}

func (port *Port) findFace(filter func(face *EthFace) bool) *EthFace {
	for _, face := range port.faces {
		if filter(face) {
			return face
		}
	}
	return nil
}

// FindFace(nil) returns a face with multicast address.
// FindFace(unicastAddr) returns a face with matching address.
func (port *Port) FindFace(remote net.HardwareAddr) *EthFace {
	if remote == nil {
		return port.findFace(func(face *EthFace) bool {
			return classifyMac48(face.remote) == mac48_multicast
		})
	}
	return port.findFace(func(face *EthFace) bool {
		return bytes.Compare(([]byte)(remote), ([]byte)(face.remote)) == 0
	})
}

func (port *Port) startFace(face *EthFace) error {
	if e := port.impl.Start(face); e != nil {
		return e
	}
	port.faces[face.GetFaceId()] = face
	return nil
}

func (port *Port) stopFace(face *EthFace) error {
	delete(port.faces, face.GetFaceId())
	return port.impl.Stop(face)
}

func (port *Port) ListFaces() (list []*EthFace) {
	for _, face := range port.faces {
		list = append(list, face)
	}
	return list
}
