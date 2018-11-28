package createface

import (
	"errors"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
)

type Config struct {
	EnableEth    bool // whether to enable Ethernet faces
	EthMtu       int  // Ethernet device MTU
	EthRxqFrames int  // Ethernet RX queue capacity
	EthTxqPkts   int  // Ethernet before-TX queue capacity
	EthTxqFrames int  // Ethernet after-TX queue capacity

	EnableSock    bool // whether to enable socket faces
	SockTxqPkts   int  // socket before-TX queue capacity
	SockTxqFrames int  // socket after-TX queue capacity

	EnableMock  bool // whether to enable mock faces
	MockTxqPkts int  // mock before-TX queue capacity
}

func GetDefaultConfig() (cfg Config) {
	cfg.EnableEth = true
	cfg.EthMtu = 0 // default MTU
	cfg.EthRxqFrames = 256
	cfg.EthTxqPkts = 256
	cfg.EthTxqFrames = 256

	cfg.EnableSock = true
	cfg.SockTxqPkts = 256
	cfg.SockTxqFrames = 256

	cfg.EnableMock = false
	cfg.MockTxqPkts = 256

	return cfg
}

func (cfg Config) Verify() error {
	return nil
}

type ICallbacks interface {
	// Callback when face mempools are needed.
	CreateFaceMempools(numaSocket dpdk.NumaSocket) (iface.Mempools, error)

	// Callback when RX mempool is needed.
	// mtu '-1' means unspecified.
	CreateRxMp(mtu int, numaSocket dpdk.NumaSocket) (dpdk.PktmbufPool, error)

	// Callback when a new RxGroup should be added.
	StartRxg(rxl iface.IRxGroup) (usr interface{}, e error)

	// Callback when an RxGroup should be removed.
	StopRxg(rxl iface.IRxGroup, usr interface{})

	// Callback when a new TxLoop should be launched.
	StartTxl(txl *iface.TxLoop) (usr interface{}, e error)

	// Callback when a TxLoop is no longer needed.
	StopTxl(txl *iface.TxLoop, usr interface{})
}

var isInitialized bool
var theConfig Config
var theCallbacks ICallbacks

func Init(cfg Config, callbacks ICallbacks) error {
	if isInitialized {
		return errors.New("already initialized")
	}

	if e := cfg.Verify(); e != nil {
		return e
	}

	theConfig = cfg
	theCallbacks = callbacks
	isInitialized = true
	return nil
}
