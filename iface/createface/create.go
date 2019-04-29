package createface

import (
	"errors"
	"fmt"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/ethface"
	"ndn-dpdk/iface/mockface"
	"ndn-dpdk/iface/socketface"
	"ndn-dpdk/ndn"
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
	if e = ctx.Launch(); e != nil {
		return nil, e
	}
	return ctx.Faces, nil
}

type createContext struct {
	Faces []iface.IFace
	eth   map[dpdk.EthDev]*createContextEth
}

type createContextEth struct {
	ethface.PortConfig
	multicastIndex int
	unicastIndex   []int
}

func newCreateContext(count int) (ctx createContext) {
	ctx.Faces = make([]iface.IFace, count)
	ctx.eth = make(map[dpdk.EthDev]*createContextEth)
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

func (ctx *createContext) addEth(i int, loc ethface.Locator) error {
	if !theConfig.EnableEth {
		return errors.New("Ethernet face feature is disabled")
	}

	isMulticast := loc.IsRemoteMulticast()
	if isMulticast && loc.Remote.String() != ndn.GetEtherMcastAddr().String() {
		return errors.New("remote MAC address must be either unicast or well-known NDN multicast group")
	}

	ethdev := dpdk.FindEthDev(loc.Port)
	if ethdev == dpdk.ETHDEV_INVALID {
		return errors.New("EthDev not found")
	}
	if ethface.FindPort(ethdev) != nil {
		return errors.New("EthDev is already active")
	}

	ectx := ctx.eth[ethdev]
	if ectx == nil {
		ectx = new(createContextEth)
		ectx.EthDev = ethdev
		ectx.Local = loc.Local
		ectx.multicastIndex = -1
		ctx.eth[ethdev] = ectx
	}

	if ectx.Local.String() != loc.Local.String() {
		return errors.New("conflicting local MAC address")
	}
	if isMulticast {
		if ectx.Multicast {
			return errors.New("EthDev already has multicast face")
		}
		ectx.Multicast = true
		ectx.multicastIndex = i
	} else {
		ectx.Unicast = append(ectx.Unicast, loc.Remote)
		ectx.unicastIndex = append(ectx.unicastIndex, i)
	}

	return nil
}

func (ctx *createContext) addSock(i int, loc socketface.Locator) (e error) {
	if !theConfig.EnableSock {
		return errors.New("socket face feature is disabled")
	}

	var cfg socketface.Config
	if cfg.Mempools, e = theCallbacks.CreateFaceMempools(dpdk.NUMA_SOCKET_ANY); e != nil {
		return e
	}
	if cfg.RxMp, e = theCallbacks.CreateRxMp(-1, dpdk.NUMA_SOCKET_ANY); e != nil {
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

func (ctx *createContext) Launch() error {
	for ethdev, ectx := range ctx.eth {
		e := ctx.launchEth(ectx)
		if e != nil {
			return fmt.Errorf("eth[%s]: %v", ethdev.GetName(), e)
		}
	}
	return nil
}

func (ctx *createContext) launchEth(ectx *createContextEth) (e error) {
	numaSocket := ectx.EthDev.GetNumaSocket()

	cfg := ectx.PortConfig
	if cfg.Mempools, e = theCallbacks.CreateFaceMempools(numaSocket); e != nil {
		return e
	}
	if cfg.RxMp, e = theCallbacks.CreateRxMp(-1, numaSocket); e != nil {
		return e
	}
	cfg.NRxThreads = 1
	cfg.RxqFrames = theConfig.EthRxqFrames
	cfg.TxqPkts = theConfig.EthTxqPkts
	cfg.TxqFrames = theConfig.EthTxqFrames
	cfg.Mtu = theConfig.EthMtu

	port, e := ethface.NewPort(cfg)
	if e != nil {
		return e
	}

	if e := startEthRxtx(port); e != nil {
		return e
	}

	if ectx.multicastIndex >= 0 {
		face := port.GetMulticastFace()
		ctx.Faces[ectx.multicastIndex] = face
	}
	for i, face := range port.ListUnicastFaces() {
		ctx.Faces[ectx.unicastIndex[i]] = face
	}
	return nil
}
