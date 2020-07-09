package createface

import (
	"errors"
	"sync"

	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/iface/ethface"
	"github.com/usnistgov/ndn-dpdk/iface/socketface"
)

var createDestroyLock sync.Mutex

// Create a face with given locator.
func Create(loc iface.Locator) (face iface.Face, e error) {
	if e = loc.Validate(); e != nil {
		return nil, e
	}
	createDestroyLock.Lock()
	defer createDestroyLock.Unlock()

	switch loc.GetScheme() {
	case "ether":
		return createEth(loc.(ethface.Locator))
	}
	return createSock(loc.(socketface.Locator))
}

func createEth(loc ethface.Locator) (face iface.Face, e error) {
	if !theConfig.EnableEth {
		return nil, errors.New("Ethernet face feature is disabled")
	}

	dev := ethdev.Find(loc.Port)
	if !dev.Valid() {
		return nil, errors.New("EthDev not found")
	}

	var cfg ethface.PortConfig
	cfg.RxqFrames = theConfig.EthRxqFrames
	cfg.TxqPkts = theConfig.EthTxqPkts
	cfg.TxqFrames = theConfig.EthTxqFrames
	cfg.Mtu = theConfig.EthMtu
	return ethface.Create(loc, cfg)
}

func createSock(loc socketface.Locator) (face iface.Face, e error) {
	if !theConfig.EnableSock {
		return nil, errors.New("socket face feature is disabled")
	}

	var cfg socketface.Config
	cfg.TxqPkts = theConfig.SockTxqPkts
	cfg.TxqFrames = theConfig.SockTxqFrames
	return socketface.New(loc, cfg)
}
