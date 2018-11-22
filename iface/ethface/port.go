package ethface

/*
#include "eth-face.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"net"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

// Minimum dataroom of PortConfig.HeaderMp.
func SizeofTxHeader() int {
	return int(C.sizeof_struct_ether_hdr) + ndn.PrependLpHeader_GetHeadroom()
}

type PortConfig struct {
	iface.Mempools
	EthDev      dpdk.EthDev
	RxMp        dpdk.PktmbufPool   // mempool for received frames
	RxqCapacity int                // receive queue length in frames
	TxqCapacity int                // send queue length in frames
	Local       net.HardwareAddr   // local address, nil for hardware default
	Multicast   bool               // whether to enable multicast face
	Unicast     []net.HardwareAddr // remote addresses for unicast faces
}

// Collection of EthFaces on a DPDK EthDev.
type Port struct {
	dev       dpdk.EthDev
	multicast *EthFace
	unicast   []*EthFace
	rxg       *RxGroup
}

var portByEthDev = make(map[dpdk.EthDev]*Port)

func FindPort(ethdev dpdk.EthDev) *Port {
	return portByEthDev[ethdev]
}

func NewPort(cfg PortConfig) (port *Port, e error) {
	if FindPort(cfg.EthDev) != nil {
		return nil, errors.New("cfg.EthDev matches existing Port")
	}
	if cfg.Local != nil && len(cfg.Local) != 6 {
		return nil, errors.New("cfg.Local is invalid")
	}
	if !cfg.Multicast && len(cfg.Unicast) == 0 {
		return nil, errors.New("cfg declares no face")
	}
	if cfg.HeaderMp.GetDataroom() < SizeofTxHeader() {
		return nil, errors.New("cfg.HeaderMp dataroom is too small")
	}

	unicastByLastOctet := make(map[byte]int)
	for i, addr := range cfg.Unicast {
		if len(addr) != 6 {
			return nil, fmt.Errorf("cfg.Unicast[%d] is invalid", i)
		}
		if j, ok := unicastByLastOctet[addr[5]]; ok {
			return nil, fmt.Errorf("cfg.Unicast[%d] has same last octet with cfg.Unicast[%d]", i, j)
		}
		unicastByLastOctet[addr[5]] = i
	}

	port = new(Port)
	port.dev = cfg.EthDev
	if e := port.startEthDev(cfg, 1); e != nil {
		return nil, e
	}

	var f faceFactory
	f.port = port
	f.mempools = cfg.Mempools
	f.local = port.dev.GetMacAddr()
	if cfg.Local != nil {
		f.local = append(net.HardwareAddr{}, cfg.Local...)
	}
	f.mtu = port.dev.GetMtu()

	if cfg.Multicast {
		faceId := iface.AllocId(iface.FaceKind_Eth)
		port.multicast = f.newFace(faceId, nil)
	}
	for i, faceId := range iface.AllocIds(iface.FaceKind_Eth, len(cfg.Unicast)) {
		face := f.newFace(faceId, cfg.Unicast[i])
		port.unicast = append(port.unicast, face)
	}

	port.rxg = newRxGroup(port, 0, 0)

	portByEthDev[cfg.EthDev] = port
	return port, nil
}

func (port *Port) startEthDev(portCfg PortConfig, nRxThreads int) error {
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
	if _, _, e := port.dev.Configure(cfg); e != nil {
		return fmt.Errorf("EthDev(%d).Configure: %v", port.dev, e)
	}

	port.dev.SetPromiscuous(true)

	if e := port.dev.Start(); e != nil {
		return fmt.Errorf("EthDev(%d).Start: %v", port.dev, e)
	}

	return nil
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
	return nil
}

func (port *Port) GetEthDev() dpdk.EthDev {
	return port.dev
}

func (port *Port) ListRxGroups() []iface.IRxGroup {
	return []iface.IRxGroup{port.rxg}
}

func (port *Port) GetNumaSocket() dpdk.NumaSocket {
	return port.dev.GetNumaSocket()
}

func (port *Port) GetMulticastFace() *EthFace {
	return port.multicast
}

func (port *Port) ListUnicastFaces() []*EthFace {
	return append([]*EthFace{}, port.unicast...)
}
