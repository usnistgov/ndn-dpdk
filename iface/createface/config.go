package createface

import (
	"errors"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/ethface"
)

type Config struct {
	EnableEth        bool // whether to enable Ethernet faces
	EthDisableRxFlow bool // whether to disable RxFlow dispatching
	EthMtu           int  // Ethernet device MTU
	EthRxqFrames     int  // Ethernet RX queue capacity
	EthTxqPkts       int  // Ethernet before-TX queue capacity
	EthTxqFrames     int  // Ethernet after-TX queue capacity

	EnableSock    bool // whether to enable socket faces
	SockTxqPkts   int  // socket before-TX queue capacity
	SockTxqFrames int  // socket after-TX queue capacity

	EnableMock bool // whether to enable mock faces

	ChanRxgFrames int // ChanRxGroup queue capacity
}

func GetDefaultConfig() (cfg Config) {
	cfg.EnableEth = true
	cfg.EthDisableRxFlow = false
	cfg.EthMtu = 0 // default MTU
	cfg.EthRxqFrames = 4096
	cfg.EthTxqPkts = 256
	cfg.EthTxqFrames = 4096

	cfg.EnableSock = true
	cfg.SockTxqPkts = 256
	cfg.SockTxqFrames = 1024

	cfg.EnableMock = false

	cfg.ChanRxgFrames = 4096

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

	ethface.DisableRxFlow = cfg.EthDisableRxFlow
	iface.TheChanRxGroup.SetQueueCapacity(cfg.ChanRxgFrames)
	return nil
}
