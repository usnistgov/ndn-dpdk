package createface

import (
	"errors"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/ethface"
	"ndn-dpdk/iface/mockface"
	"ndn-dpdk/iface/socketface"
)

func Create(loc iface.Locator) (face iface.IFace, e error) {
	if theConfig.Disabled {
		return nil, errors.New("createface package is disabled")
	}
	createDestroyLock.Lock()
	defer createDestroyLock.Unlock()

	ctx := newCreateContext(1)
	if e = ctx.Add(0, loc); e != nil {
		return nil, e
	}
	return ctx.Faces[0], nil
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

func (ctx *createContext) addEth(i int, loc ethface.Locator) (e error) {
	if !theConfig.EnableEth {
		return errors.New("Ethernet face feature is disabled")
	}

	dev := dpdk.FindEthDev(loc.Port)
	if dev == dpdk.ETHDEV_INVALID {
		return errors.New("EthDev not found")
	}

	numaSocket := dev.GetNumaSocket()
	var cfg ethface.PortConfig
	if cfg.RxMp, cfg.Mempools, e = getMempools(numaSocket); e != nil {
		return e
	}
	cfg.RxqFrames = theConfig.EthRxqFrames
	cfg.TxqPkts = theConfig.EthTxqPkts
	cfg.TxqFrames = theConfig.EthTxqFrames
	cfg.Mtu = theConfig.EthMtu

	face, e := ethface.Create(loc, cfg)
	if e != nil {
		return e
	}
	ctx.Faces[i] = face
	return nil
}

func (ctx *createContext) addSock(i int, loc socketface.Locator) (e error) {
	if !theConfig.EnableSock {
		return errors.New("socket face feature is disabled")
	}

	var cfg socketface.Config
	if cfg.RxMp, cfg.Mempools, e = getMempools(dpdk.NUMA_SOCKET_ANY); e != nil {
		return e
	}
	cfg.TxqPkts = theConfig.SockTxqPkts
	cfg.TxqFrames = theConfig.SockTxqFrames

	face, e := socketface.Create(loc, cfg)
	if e != nil {
		return e
	}
	ctx.Faces[i] = face
	return nil
}

var hasMockFaces = false

func (ctx *createContext) addMock(i int) (e error) {
	if !theConfig.EnableMock {
		return errors.New("mock face feature is disabled")
	}
	if !hasMockFaces {
		if _, mockface.FaceMempools, e = getMempools(dpdk.NUMA_SOCKET_ANY); e != nil {
			return e
		}
		hasMockFaces = true
	}

	face := mockface.New()
	ctx.Faces[i] = face
	return nil
}
