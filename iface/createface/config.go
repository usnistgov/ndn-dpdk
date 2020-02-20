package createface

import (
	"errors"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/ethface"
)

var (
	theConfig   Config
	theMempools = make(map[dpdk.NumaSocket]numaMempools)
	theRxls     []*iface.RxLoop
	theTxls     []*iface.TxLoop

	CustomGetRxl func(rxg iface.IRxGroup) *iface.RxLoop
	CustomGetTxl func(rxg iface.IFace) *iface.TxLoop
)

type Config struct {
	Disabled bool // whether to disable this package

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
	cfg.Disabled = false

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
	if cfg.EnableEth {
		if cfg.EthRxqFrames < 64 {
			return errors.New("cfg.EthRxqFrames must be at least 64")
		}
		if cfg.EthTxqPkts < 64 {
			return errors.New("cfg.EthTxqPkts must be at least 64")
		}
		if cfg.EthTxqFrames < 64 {
			return errors.New("cfg.EthTxqFrames must be at least 64")
		}
	}
	if cfg.EnableSock {
		if cfg.SockTxqPkts < 64 {
			return errors.New("cfg.SockTxqPkts must be at least 64")
		}
		if cfg.SockTxqFrames < 64 {
			return errors.New("cfg.SockTxqFrames must be at least 64")
		}
	}
	if cfg.EnableSock || cfg.EnableMock {
		if cfg.ChanRxgFrames < 64 {
			return errors.New("cfg.ChanRxgFrames must be at least 64")
		}
	}
	return nil
}

func (cfg Config) Apply() error {
	if e := cfg.Verify(); e != nil {
		return e
	}
	theConfig = cfg
	ethface.DisableRxFlow = cfg.EthDisableRxFlow
	iface.TheChanRxGroup.SetQueueCapacity(cfg.ChanRxgFrames)
	return nil
}

// List NumaSockets for RxLoops and TxLoops to satisfy enabled devices.
func ListRxTxNumaSockets() (list []dpdk.NumaSocket) {
	if theConfig.EnableEth {
		for _, port := range dpdk.ListEthDevs() {
			list = append(list, port.GetNumaSocket())
		}
	}
	if theConfig.EnableSock || theConfig.EnableMock {
		list = append(list, dpdk.NUMA_SOCKET_ANY)
	}
	return list
}

type numaMempools struct {
	RxMp         dpdk.PktmbufPool
	FaceMempools iface.Mempools
}

// Provide a set of mempools for face creation.
func AddMempools(numaSocket dpdk.NumaSocket, rxMp dpdk.PktmbufPool, faceMempools iface.Mempools) {
	theMempools[numaSocket] = numaMempools{
		RxMp:         rxMp,
		FaceMempools: faceMempools,
	}
}

func getMempools(numaSocket dpdk.NumaSocket) (rxMp dpdk.PktmbufPool, faceMempools iface.Mempools, e error) {
	// allocate from preferred NumaSocket
	if numaMp, ok := theMempools[numaSocket]; ok {
		return numaMp.RxMp, numaMp.FaceMempools, nil
	}

	// allocate from any NumaSocket
	if numaSocket != dpdk.NUMA_SOCKET_ANY {
		return getMempools(dpdk.NUMA_SOCKET_ANY)
	}

	// allocate from other NumaSocket
	for _, numaMp := range theMempools {
		return numaMp.RxMp, numaMp.FaceMempools, nil
	}

	// fail
	return dpdk.PktmbufPool{}, iface.Mempools{}, errors.New("mempools unavailable")
}

// Provide an RxLoop for face creation.
func AddRxLoop(rxl *iface.RxLoop) {
	theRxls = append(theRxls, rxl)
}

// Provide a TxLoop for face creation.
func AddTxLoop(txl *iface.TxLoop) {
	theTxls = append(theTxls, txl)
}

// Close all faces and stop RxLoops and TxLoops.
func CloseAll() (threads []dpdk.IThread) {
	iface.CloseAll()
	for _, rxl := range theRxls {
		rxl.Stop()
		rxl.Close()
		threads = append(threads, rxl)
	}
	theRxls = nil
	for _, txl := range theTxls {
		txl.Stop()
		txl.Close()
		threads = append(threads, txl)
	}
	theTxls = nil
	return threads
}
