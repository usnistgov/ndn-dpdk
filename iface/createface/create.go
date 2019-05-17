package createface

import (
	"errors"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/ethface"
	"ndn-dpdk/iface/mockface"
	"ndn-dpdk/iface/socketface"
)

// Create a face with given locator.
func Create(loc iface.Locator) (face iface.IFace, e error) {
	if theConfig.Disabled {
		return nil, errors.New("createface package is disabled")
	}
	if e = loc.Validate(); e != nil {
		return nil, e
	}
	createDestroyLock.Lock()
	defer createDestroyLock.Unlock()

	switch loc.GetScheme() {
	case "ether":
		return createEth(loc.(ethface.Locator))
	case "mock":
		return createMock()
	}
	return createSock(loc.(socketface.Locator))
}

func createEth(loc ethface.Locator) (face iface.IFace, e error) {
	if !theConfig.EnableEth {
		return nil, errors.New("Ethernet face feature is disabled")
	}

	dev := dpdk.FindEthDev(loc.Port)
	if dev == dpdk.ETHDEV_INVALID {
		return nil, errors.New("EthDev not found")
	}

	numaSocket := dev.GetNumaSocket()
	var cfg ethface.PortConfig
	if cfg.RxMp, cfg.Mempools, e = getMempools(numaSocket); e != nil {
		return nil, e
	}
	cfg.RxqFrames = theConfig.EthRxqFrames
	cfg.TxqPkts = theConfig.EthTxqPkts
	cfg.TxqFrames = theConfig.EthTxqFrames
	cfg.Mtu = theConfig.EthMtu
	return ethface.Create(loc, cfg)
}

func createSock(loc socketface.Locator) (face iface.IFace, e error) {
	if !theConfig.EnableSock {
		return nil, errors.New("socket face feature is disabled")
	}

	var cfg socketface.Config
	if cfg.RxMp, cfg.Mempools, e = getMempools(dpdk.NUMA_SOCKET_ANY); e != nil {
		return nil, e
	}
	cfg.TxqPkts = theConfig.SockTxqPkts
	cfg.TxqFrames = theConfig.SockTxqFrames
	return socketface.Create(loc, cfg)
}

var hasMockFaces = false

func createMock() (face iface.IFace, e error) {
	if !theConfig.EnableMock {
		return nil, errors.New("mock face feature is disabled")
	}

	if !hasMockFaces {
		if _, mockface.FaceMempools, e = getMempools(dpdk.NUMA_SOCKET_ANY); e != nil {
			return nil, e
		}
		hasMockFaces = true
	}

	return mockface.New(), nil
}
