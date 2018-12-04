package ethface

/*
#include "eth-face.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"net"

	"github.com/sirupsen/logrus"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/faceuri"
	"ndn-dpdk/ndn"
)

// Minimum dataroom of PortConfig.HeaderMp.
func SizeofTxHeader() int {
	return int(C.sizeof_struct_ether_hdr) + ndn.PrependLpHeader_GetHeadroom()
}

// Port creation arguments.
type PortConfig struct {
	iface.Mempools
	EthDev      dpdk.EthDev
	RxMp        dpdk.PktmbufPool   // mempool for received frames
	RxqCapacity int                // receive queue length in frames
	TxqCapacity int                // send queue length in frames
	Mtu         int                // set MTU, 0 to keep default
	Local       net.HardwareAddr   // local address, nil for hardware default
	Multicast   bool               // whether to enable multicast face
	Unicast     []net.HardwareAddr // remote addresses for unicast faces
	faceIds     []iface.FaceId     // assigned FaceIds
}

func (cfg PortConfig) check() error {
	if cfg.Local != nil {
		if addr := faceuri.MacAddress(cfg.Local); !addr.Valid() || addr.IsGroupAddress() {
			return errors.New("cfg.Local is not a MAC-48 unicast address")
		}
	}

	if cfg.countFaces() == 0 {
		return errors.New("cfg declares no face")
	}

	unicastAddressStr := make(map[string]int)
	for i, unicastAddr := range cfg.Unicast {
		addr := faceuri.MacAddress(unicastAddr)
		if !addr.Valid() || addr.IsGroupAddress() {
			return fmt.Errorf("cfg.Unicast[%d] is not a MAC-48 unicast address", i)
		}
		if j, ok := unicastAddressStr[addr.String()]; ok {
			return fmt.Errorf("cfg.Unicast[%d] duplicates cfg.Unicast[%d]", i, j)
		}
		unicastAddressStr[addr.String()] = i
	}

	if cfg.HeaderMp.GetDataroom() < SizeofTxHeader() {
		return errors.New("cfg.HeaderMp dataroom is too small")
	}

	return nil
}

func (cfg PortConfig) countFaces() int {
	nFaces := len(cfg.Unicast)
	if cfg.Multicast {
		nFaces++
	}
	return nFaces
}

func (cfg *PortConfig) allocIds() {
	cfg.faceIds = iface.AllocIds(iface.FaceKind_Eth, cfg.countFaces())
}

func (cfg PortConfig) getFaceIdAddr(i int) (id iface.FaceId, addr net.HardwareAddr) {
	if cfg.faceIds != nil {
		id = cfg.faceIds[i]
	} else {
		id = iface.FACEID_INVALID
	}

	if i == len(cfg.Unicast) {
		return id, nil
	}
	return id, cfg.Unicast[i]
}

// Collection of EthFaces on a DPDK EthDev.
type Port struct {
	dev       dpdk.EthDev
	logger    logrus.FieldLogger
	multicast *EthFace
	unicast   []*EthFace
	rxt       *RxTable
}

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

func NewPort(cfg PortConfig) (port *Port, e error) {
	if e = cfg.check(); e != nil {
		return nil, e
	}
	if FindPort(cfg.EthDev) != nil {
		return nil, errors.New("cfg.EthDev matches existing Port")
	}
	cfg.allocIds()

	port = new(Port)
	port.logger = newPortLogger(cfg.EthDev)
	port.dev = cfg.EthDev
	rxgErr := make(rxgStartErrors, 0)
	for _, rxgStarter := range rxgStarters {
		cfg.EthDev.Reset()
		name := rxgStarter.String()
		e = rxgStarter.Start(port, cfg)
		if e == nil {
			port.logger.WithField("rxg", name).Info("started")
			rxgErr = nil
			break
		} else {
			rxgErr = append(rxgErr, rxgStartError{name, e})
		}
	}
	if rxgErr != nil {
		port.logger.WithError(rxgErr).Error("no RxGroup impl available")
		return nil, rxgErr
	}

	portByEthDev[cfg.EthDev] = port
	return port, nil
}

func (port *Port) configureDev(portCfg PortConfig, nRxThreads int) error {
	var cfg dpdk.EthDevConfig
	numaSocket := port.GetNumaSocket()
	for i := 0; i < nRxThreads; i++ {
		cfg.AddRxQueue(dpdk.EthRxQueueConfig{
			Capacity: portCfg.RxqCapacity,
			Socket:   numaSocket,
			Mp:       portCfg.RxMp,
		})
	}
	cfg.AddTxQueue(dpdk.EthTxQueueConfig{
		Capacity: portCfg.TxqCapacity,
		Socket:   numaSocket,
	})
	cfg.Mtu = portCfg.Mtu
	if _, _, e := port.dev.Configure(cfg); e != nil {
		return fmt.Errorf("EthDev(%d).Configure: %v", port.dev, e)
	}
	return nil
}

func (port *Port) startDev() error {
	if e := port.dev.Start(); e != nil {
		return fmt.Errorf("EthDev(%d).Start: %v", port.dev, e)
	}
	return nil
}

func (port *Port) createFaces(cfg PortConfig, flows map[iface.FaceId]*RxFlow) {
	var f faceFactory
	f.port = port
	f.mempools = cfg.Mempools
	f.local = port.dev.GetMacAddr()
	if cfg.Local != nil {
		f.local = append(net.HardwareAddr{}, cfg.Local...)
	}
	f.mtu = port.dev.GetMtu()
	f.flows = flows

	for i, nFaces := 0, cfg.countFaces(); i < nFaces; i++ {
		id, addr := cfg.getFaceIdAddr(i)
		face := f.newFace(id, addr)
		if addr == nil {
			port.multicast = face
		} else {
			port.unicast = append(port.unicast, face)
		}
	}
}

func (port *Port) Close() error {
	if port.multicast != nil {
		port.multicast.Close()
	}
	for _, face := range port.unicast {
		face.Close()
	}
	port.dev.Stop()
	delete(portByEthDev, port.dev)
	port.logger.Info("closed")
	return nil
}

func (port *Port) GetEthDev() dpdk.EthDev {
	return port.dev
}

func (port *Port) ListRxGroups() (list []iface.IRxGroup) {
	if port.rxt != nil {
		return []iface.IRxGroup{port.rxt}
	}

	if port.multicast != nil {
		list = append(list, port.multicast.rxf)
	}
	for _, face := range port.unicast {
		list = append(list, face.rxf)
	}
	return list
}

func (port *Port) GetNumaSocket() dpdk.NumaSocket {
	return port.dev.GetNumaSocket()
}

func (port *Port) CountFaces() int {
	n := len(port.unicast)
	if port.multicast != nil {
		n++
	}
	return n
}

func (port *Port) GetMulticastFace() *EthFace {
	return port.multicast
}

func (port *Port) ListUnicastFaces() []*EthFace {
	return append([]*EthFace{}, port.unicast...)
}
