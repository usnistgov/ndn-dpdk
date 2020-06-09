package ethface

import (
	"fmt"
	"io"

	"ndn-dpdk/dpdk"
)

// RX/TX setup implementation.
type iImpl interface {
	fmt.Stringer
	io.Closer

	// Construct new instance.
	New(port *Port) iImpl

	// Initialize.
	Init() error

	// Start a face.
	Start(face *EthFace) error

	// Stop a face.
	Stop(face *EthFace) error
}

var impls = []iImpl{&rxFlowImpl{}, &rxTableImpl{}}

// Start EthDev (called by impl).
func startDev(port *Port, nRxQueues int, promisc bool) error {
	var cfg dpdk.EthDevConfig
	numaSocket := port.dev.GetNumaSocket()
	for i := 0; i < nRxQueues; i++ {
		cfg.RxQueues = append(cfg.RxQueues, dpdk.EthRxQueueConfig{
			Capacity: port.cfg.RxqFrames,
			Socket:   numaSocket,
			Mp:       port.cfg.RxMp,
		})
	}
	cfg.TxQueues = append(cfg.TxQueues, dpdk.EthTxQueueConfig{
		Capacity: port.cfg.TxqFrames,
		Socket:   numaSocket,
	})
	cfg.Mtu = port.cfg.Mtu
	if _, _, e := port.dev.Configure(cfg); e != nil {
		return e
	}

	if promisc {
		port.dev.SetPromiscuous(true)
	}
	return port.dev.Start()
}
