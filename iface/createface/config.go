package createface

import (
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/dpdk/ringbuffer"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/iface/ethface"
	"github.com/usnistgov/ndn-dpdk/iface/socketface"
)

var (
	theConfig Config
	theRxls   []iface.RxLoop
	theTxls   []iface.TxLoop

	CustomGetRxl func(rxg iface.RxGroup) iface.RxLoop
	CustomGetTxl func(rxg iface.Face) iface.TxLoop
)

type Config struct {
	EnableEth        bool // whether to enable Ethernet faces
	EthDisableRxFlow bool // whether to disable RxFlow dispatching
	EthMtu           int  // Ethernet device MTU
	EthRxqFrames     int  // Ethernet RX queue capacity
	EthTxqPkts       int  // Ethernet before-TX queue capacity
	EthTxqFrames     int  // Ethernet after-TX queue capacity

	EnableSock    bool // whether to enable socket faces
	SockRxqFrames int  // socket RX queue capacity (shared)
	SockTxqPkts   int  // socket before-TX queue capacity
	SockTxqFrames int  // socket after-TX queue capacity
}

func (cfg Config) Apply() {
	cfg.EthRxqFrames = ringbuffer.AlignCapacity(cfg.EthRxqFrames, 64, 4096)
	cfg.EthTxqPkts = ringbuffer.AlignCapacity(cfg.EthTxqPkts, 64, 256)
	cfg.EthTxqFrames = ringbuffer.AlignCapacity(cfg.EthTxqFrames, 64, 4096)
	cfg.SockRxqFrames = ringbuffer.AlignCapacity(cfg.SockRxqFrames, 64, 4096)
	cfg.SockTxqPkts = ringbuffer.AlignCapacity(cfg.SockTxqPkts, 64, 256)
	cfg.SockTxqFrames = ringbuffer.AlignCapacity(cfg.SockTxqFrames, 64, 4096)

	theConfig = cfg
	ethface.DisableRxFlow = cfg.EthDisableRxFlow
	socketface.ChangeRxQueueCapacity(cfg.SockRxqFrames)
}

// List NumaSockets for RxLoops and TxLoops to satisfy enabled devices.
func ListRxTxNumaSockets() (list []eal.NumaSocket) {
	if theConfig.EnableEth {
		for _, port := range ethdev.List() {
			list = append(list, port.NumaSocket())
		}
	}
	if theConfig.EnableSock {
		list = append(list, eal.NumaSocket{})
	}
	return list
}

// Provide an RxLoop for face creation.
func AddRxLoop(rxl iface.RxLoop) {
	theRxls = append(theRxls, rxl)
}

// Provide a TxLoop for face creation.
func AddTxLoop(txl iface.TxLoop) {
	theTxls = append(theTxls, txl)
}

// Close all faces and stop RxLoops and TxLoops.
func CloseAll() {
	iface.CloseAll()
	for _, rxl := range theRxls {
		rxl.Close()
	}
	theRxls = nil
	for _, txl := range theTxls {
		txl.Close()
	}
	theTxls = nil
}
