package createface

import (
	"errors"
	"fmt"
	"net"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/ethface"
	"ndn-dpdk/iface/mockface"
	"ndn-dpdk/iface/socketface"
)

func Create(locs ...iface.Locator) (faces []iface.IFace, e error) {
	if !isInitialized {
		return nil, errors.New("facecreate package is uninitialized")
	}

	createDestroyLock.Lock()
	defer createDestroyLock.Unlock()

	ctx := newCreateContext(len(locs))
	for i, loc := range locs {
		if e = ctx.Add(i, loc); e != nil {
			return nil, fmt.Errorf("loc[%d]: %v", i, e)
		}
	}
	return ctx.Faces, nil
}

type createContext struct {
	Faces []iface.IFace
}

func newCreateContext(count int) (ctx createContext) {
	ctx.Faces = make([]iface.IFace, count)
	return ctx
}

func (ctx *createContext) Add(i int, loc iface.Locator) error {
	if e := loc.Validate(); e != nil {
		return e
	}

	switch loc.GetScheme() {
	case "ether":
		return ctx.addEth(i, loc.(ethface.Locator))
	case "mock":
		return ctx.addMock(i)
	}
	return ctx.addSock(i, loc.(socketface.Locator))
}

func (ctx *createContext) createEthPort(dev dpdk.EthDev, local net.HardwareAddr) (port *ethface.Port, e error) {
	numaSocket := dev.GetNumaSocket()
	var cfg ethface.PortConfig
	cfg.EthDev = dev
	if cfg.Mempools, e = theCallbacks.CreateFaceMempools(numaSocket); e != nil {
		return nil, e
	}
	if cfg.RxMp, e = theCallbacks.CreateRxMp(numaSocket); e != nil {
		return nil, e
	}
	cfg.RxqFrames = theConfig.EthRxqFrames
	cfg.TxqPkts = theConfig.EthTxqPkts
	cfg.TxqFrames = theConfig.EthTxqFrames
	cfg.Mtu = theConfig.EthMtu
	cfg.Local = local
	return ethface.NewPort(cfg)
}

func (ctx *createContext) addEth(i int, loc ethface.Locator) (e error) {
	if !theConfig.EnableEth {
		return errors.New("Ethernet face feature is disabled")
	}

	dev := dpdk.FindEthDev(loc.Port)
	if dev == dpdk.ETHDEV_INVALID {
		return errors.New("EthDev not found")
	}

	port := ethface.FindPort(dev)
	if port == nil {
		if port, e = ctx.createEthPort(dev, loc.Local); e != nil {
			return e
		}
	}

	face, e := ethface.New(port, loc)
	if e != nil {
		return e
	}
	ctx.Faces[i] = face
	return startEthRxtx(face)
}

func (ctx *createContext) addSock(i int, loc socketface.Locator) (e error) {
	if !theConfig.EnableSock {
		return errors.New("socket face feature is disabled")
	}

	var cfg socketface.Config
	if cfg.Mempools, e = theCallbacks.CreateFaceMempools(dpdk.NUMA_SOCKET_ANY); e != nil {
		return e
	}
	if cfg.RxMp, e = theCallbacks.CreateRxMp(dpdk.NUMA_SOCKET_ANY); e != nil {
		return e
	}
	cfg.TxqPkts = theConfig.SockTxqPkts
	cfg.TxqFrames = theConfig.SockTxqFrames

	face, e := socketface.Create(loc, cfg)
	if e != nil {
		return e
	}
	ctx.Faces[i] = face
	return startChanRxtx(face)
}

var hasMockFaces = false

func (ctx *createContext) addMock(i int) (e error) {
	if !theConfig.EnableMock {
		return errors.New("mock face feature is disabled")
	}
	if !hasMockFaces {
		if mockface.FaceMempools, e = theCallbacks.CreateFaceMempools(dpdk.NUMA_SOCKET_ANY); e != nil {
			return e
		}
		hasMockFaces = true
	}

	face := mockface.New()
	ctx.Faces[i] = face
	return startChanRxtx(face)
}
